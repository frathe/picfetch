// Qualification app: its own container is private from the worker service.
// A temporary bookmark transfers only one fixture file, without a file picker.
#import <Foundation/Foundation.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

int main(int argc, char **argv) {
    @autoreleasepool {
        if (argc != 2) return 2;
        NSString *broker = [NSBundle.mainBundle.bundlePath
            stringByAppendingPathComponent:@"Contents/MacOS/picfetch-worker-client"];
        const char *mode = argv[1];
        if (!strcmp(mode, "grant")) {
            NSURL *directory = [NSFileManager.defaultManager URLsForDirectory:NSDocumentDirectory
                inDomains:NSUserDomainMask].firstObject;
            if (![NSFileManager.defaultManager createDirectoryAtURL:directory
                withIntermediateDirectories:YES attributes:nil error:NULL]) return 4;
            NSURL *source = [directory URLByAppendingPathComponent:@"picfetch-grant-fixture.dat"];
            NSURL *sibling = [directory URLByAppendingPathComponent:@"picfetch-ungranted-fixture.dat"];
            NSData *content = [@"fixture" dataUsingEncoding:NSUTF8StringEncoding];
            if (![content writeToURL:source options:NSDataWritingAtomic error:NULL] ||
                ![content writeToURL:sibling options:NSDataWritingAtomic error:NULL]) return 4;
            NSData *bookmark = [source bookmarkDataWithOptions:0 includingResourceValuesForKeys:nil
                relativeToURL:nil error:NULL];
            if (!bookmark) return 4;
            NSData *payload = [NSJSONSerialization dataWithJSONObject:@{
                @"path": source.path, @"sibling": sibling.path,
                @"bookmark": [bookmark base64EncodedStringWithOptions:0]}
                options:0 error:NULL];
            // Keep this fixture write within PIPE_BUF; production uses its
            // existing bounded request stream rather than a pre-filled pipe.
            if (!payload || payload.length + 1 > 4096) return 4;
            int input[2];
            if (pipe(input)) return 4;
            if (write(input[1], payload.bytes, payload.length) != (ssize_t)payload.length ||
                write(input[1], "\n", 1) != 1 || dup2(input[0], STDIN_FILENO) < 0) return 4;
            close(input[0]); close(input[1]);
            mode = "similarity";
        }
        char *arguments[] = {(char *)broker.fileSystemRepresentation, (char *)mode, NULL};
        execv(broker.fileSystemRepresentation, arguments);
        perror("qualification broker");
        return 4;
    }
}
