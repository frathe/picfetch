// Each connection owns one child process group. Its leader is not reaped until
// remaining descendants have been killed, so cancellation never signals a PID
// that the kernel could already have reused for an unrelated process.
#import "protocol.h"
#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <spawn.h>
#include <stdio.h>
#include <sys/wait.h>
#include <unistd.h>

extern char **environ;

@interface PicFetchJob : NSObject <PicFetchWorker>
@property(nonatomic, strong) NSLock *lock;
@property(nonatomic) pid_t child;
@property(nonatomic) BOOL submitted;
@property(nonatomic) BOOL cancelled;
@end

@implementation PicFetchJob
- (instancetype)init {
    self = [super init];
    if (self) _lock = [NSLock new];
    return self;
}
- (void)cancel {
    [self.lock lock];
    self.cancelled = YES;
    if (self.child > 0) kill(-self.child, SIGKILL);
    [self.lock unlock];
}
- (void)runMode:(NSString *)mode input:(NSFileHandle *)input
         output:(NSFileHandle *)output error:(NSFileHandle *)error
          reply:(void (^)(int))reply {
    [self.lock lock];
    if (self.submitted || self.cancelled ||
        (![mode isEqualToString:@"heic"] && ![mode isEqualToString:@"similarity"])) {
        [self.lock unlock]; reply(2); return;
    }
    self.submitted = YES;
    // Duplicating above stderr prevents dup2 ordering from clobbering another
    // transferred descriptor, including when the service began with closed stdio.
    int descriptors[] = {-1, -1, -1};
    NSFileHandle *handles[] = {input, output, error};
    for (int i = 0; i < 3; i++) {
        if (handles[i]) descriptors[i] = fcntl(handles[i].fileDescriptor, F_DUPFD_CLOEXEC, 3);
        if (descriptors[i] < 0) {
            for (int j = 0; j < i; j++) close(descriptors[j]);
            [self.lock unlock]; reply(1); return;
        }
    }
    NSURL *contents = [NSBundle.mainBundle.bundleURL URLByAppendingPathComponent:@"Contents"];
    NSString *executable = [contents URLByAppendingPathComponent:@"MacOS/picfetch-image-worker"].path;
    posix_spawn_file_actions_t actions;
    posix_spawnattr_t attributes;
    int failure = posix_spawn_file_actions_init(&actions);
    if (failure) { for (int i = 0; i < 3; i++) close(descriptors[i]); [self.lock unlock]; reply(1); return; }
    failure = posix_spawnattr_init(&attributes);
    if (failure) {
        posix_spawn_file_actions_destroy(&actions);
        for (int i = 0; i < 3; i++) close(descriptors[i]);
        [self.lock unlock]; reply(1); return;
    }
    if (!failure) failure = posix_spawn_file_actions_adddup2(&actions, descriptors[0], STDIN_FILENO);
    if (!failure) failure = posix_spawn_file_actions_adddup2(&actions, descriptors[1], STDOUT_FILENO);
    if (!failure) failure = posix_spawn_file_actions_adddup2(&actions, descriptors[2], STDERR_FILENO);
    if (!failure) failure = posix_spawnattr_setflags(&attributes, POSIX_SPAWN_SETPGROUP | POSIX_SPAWN_CLOEXEC_DEFAULT);
    if (!failure) failure = posix_spawnattr_setpgroup(&attributes, 0);
    char *arguments[] = {(char *)executable.fileSystemRepresentation, NULL};
    // The worker mode is selected by this trusted service, not caller-supplied
    // environment or arguments. Source/model/cache inputs stay on the pipe.
    NSString *workerFlag = [mode isEqualToString:@"heic"] ? @"PICFETCH_HEIC_WORKER=1" : @"PICFETCH_SIMILARITY_WORKER=1";
    NSUInteger count = 0;
    while (environ[count]) count++;
    char **environment = calloc(count + 2, sizeof(char *));
    if (!environment) failure = ENOMEM;
    if (!failure) {
        NSUInteger used = 0;
        for (NSUInteger i = 0; i < count; i++) {
            if (strncmp(environ[i], "PICFETCH_HEIC_WORKER=", 20) &&
                strncmp(environ[i], "PICFETCH_SIMILARITY_WORKER=", 26)) environment[used++] = environ[i];
        }
        environment[used] = (char *)workerFlag.UTF8String;
        failure = posix_spawn(&_child, executable.fileSystemRepresentation, &actions,
            &attributes, arguments, environment);
    }
    free(environment);
    posix_spawnattr_destroy(&attributes);
    posix_spawn_file_actions_destroy(&actions);
    for (int i = 0; i < 3; i++) close(descriptors[i]);
    if (failure) { dprintf(error.fileDescriptor, "worker spawn: %s (%s)\n", strerror(failure), executable.UTF8String); [self.lock unlock]; reply(1); return; }
    pid_t child = self.child;
    [self.lock unlock];
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0), ^{
        siginfo_t info = {0};
        int result;
        do { result = waitid(P_PID, child, &info, WEXITED | WNOWAIT); } while (result < 0 && errno == EINTR);
        [self.lock lock];
        // The leader still belongs to us. Retire descendants before reaping.
        kill(-child, SIGKILL);
        int status = 0;
        pid_t reaped;
        do { reaped = waitpid(child, &status, 0); } while (reaped < 0 && errno == EINTR);
        self.child = 0;
        [self.lock unlock];
        if (result < 0 || reaped != child) dprintf(error.fileDescriptor, "worker wait: result=%d reaped=%d errno=%d\n", result, reaped, errno);
        int exitStatus = 1;
        if (result == 0 && reaped == child) {
            if (WIFEXITED(status)) exitStatus = WEXITSTATUS(status);
            else if (WIFSIGNALED(status)) exitStatus = 128 + WTERMSIG(status);
        }
        reply(exitStatus);
    });
}
@end

@interface PicFetchListener : NSObject <NSXPCListenerDelegate>
@end
@implementation PicFetchListener
- (BOOL)listener:(NSXPCListener *)listener shouldAcceptNewConnection:(NSXPCConnection *)connection {
    PicFetchJob *job = [PicFetchJob new];
    connection.exportedInterface = [NSXPCInterface interfaceWithProtocol:@protocol(PicFetchWorker)];
    connection.exportedObject = job;
    connection.invalidationHandler = ^{ [job cancel]; };
    [connection resume];
    return YES;
}
@end

int main(void) {
    @autoreleasepool {
        PicFetchListener *delegate = [PicFetchListener new];
        NSXPCListener *listener = NSXPCListener.serviceListener;
        listener.delegate = delegate;
        [listener resume];
    }
    return 0;
}
