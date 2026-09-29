// The app's inherited helper transfers its existing pipes to a network-free
// XPC service. No image paths or executable locations cross this interface.
#import "protocol.h"
#include <signal.h>
#include <stdio.h>
#include <string.h>

int main(int argc, const char **argv) {
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
        signal(SIGTERM, SIG_IGN);
        dispatch_source_t cancellation = dispatch_source_create(DISPATCH_SOURCE_TYPE_SIGNAL,
            SIGTERM, 0, dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0));
        dispatch_source_set_event_handler(cancellation, ^{
            [[connection remoteObjectProxyWithErrorHandler:^(NSError *error) {
                fprintf(stderr, "worker cancellation: %s\n", error.description.UTF8String);
                finish(1);
            }] cancel];
        });
        dispatch_resume(cancellation);
        [connection resume];
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
