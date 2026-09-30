// The app's inherited helper transfers its existing pipes to a network-free
// XPC service. No image paths or executable locations cross this interface.
#import "protocol.h"
#include <signal.h>
#include <stdio.h>
#include <string.h>

// Retain a cancellation that arrives before dispatch registers its signal source.
// The handler only records a signal-safe flag; XPC work stays on dispatch.
static volatile sig_atomic_t cancellationRequested = 0;
static void retainCancellation(int number) {
    (void)number;
    cancellationRequested = 1;
}

int main(int argc, const char **argv) {
    if (signal(SIGTERM, retainCancellation) == SIG_ERR) {
        perror("worker cancellation setup");
        return 1;
    }
    // A launcher may leave SIGTERM blocked across exec. Once the retaining
    // handler is installed it is safe to deliver that pending cancellation.
    sigset_t cancellationMask;
    sigemptyset(&cancellationMask);
    sigaddset(&cancellationMask, SIGTERM);
    if (sigprocmask(SIG_UNBLOCK, &cancellationMask, NULL) < 0) {
        perror("worker cancellation mask");
        return 1;
    }
    @autoreleasepool {
        if (argc != 2 || (strcmp(argv[1], "heic") && strcmp(argv[1], "similarity"))) {
            fprintf(stderr, "usage: picfetch-worker-client heic|similarity\n");
            return 2;
        }
        NSXPCConnection *connection = [[NSXPCConnection alloc]
            initWithServiceName:@"io.github.frathe.picfetch.worker"];
        connection.remoteObjectInterface = [NSXPCInterface interfaceWithProtocol:@protocol(PicFetchWorker)];
        dispatch_semaphore_t done = dispatch_semaphore_create(0);
        NSLock *completionLock = [NSLock new];
        __block BOOL finished = NO;
        __block int status = 1;
        void (^finish)(int) = ^(int result) {
            [completionLock lock];
            if (!finished) {
                status = result;
                finished = YES;
                dispatch_semaphore_signal(done);
            }
            [completionLock unlock];
        };
        connection.interruptionHandler = ^{ finish(1); };
        connection.invalidationHandler = ^{ finish(1); };

        // Cancellation must allow the service to kill and reap its children
        // before the caller's Wait returns. A killed broker also invalidates
        // the connection; the service independently retires its process group.
        dispatch_source_t cancellation = dispatch_source_create(DISPATCH_SOURCE_TYPE_SIGNAL,
            SIGTERM, 0, dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0));
        void (^cancel)(void) = ^{
            [[connection remoteObjectProxyWithErrorHandler:^(NSError *error) {
                fprintf(stderr, "worker cancellation: %s\n", error.description.UTF8String);
                finish(1);
            }] cancel];
        };
        dispatch_source_set_event_handler(cancellation, cancel);
        dispatch_source_set_registration_handler(cancellation, ^{
            if (cancellationRequested) cancel();
        });
        [connection resume];
        dispatch_resume(cancellation);
        [[connection remoteObjectProxyWithErrorHandler:^(NSError *error) {
            fprintf(stderr, "worker service: %s\n", error.description.UTF8String);
            finish(1);
        }] runMode:[NSString stringWithUTF8String:argv[1]]
            input:[NSFileHandle fileHandleWithStandardInput]
            output:[NSFileHandle fileHandleWithStandardOutput]
            error:[NSFileHandle fileHandleWithStandardError]
            reply:^(int result) { finish(result); }];
        dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
        dispatch_source_cancel(cancellation);
        [connection invalidate];
        [completionLock lock];
        int result = status;
        [completionLock unlock];
        return result;
    }
}
