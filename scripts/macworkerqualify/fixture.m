// Qualification-only worker: no application data or model is used.
#import <Foundation/Foundation.h>
#include <arpa/inet.h>
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

static int denial(int type) {
    int fd = socket(AF_INET, type, 0);
    if (fd < 0) return errno;
    if (fcntl(fd, F_SETFL, O_NONBLOCK) < 0) {
        int failure = errno; close(fd); return failure;
    }
    struct sockaddr_in address = {0};
    address.sin_family = AF_INET;
    address.sin_port = htons(443);
    inet_pton(AF_INET, "192.0.2.1", &address.sin_addr);
    int result = connect(fd, (struct sockaddr *)&address, sizeof address);
    int failure = result < 0 ? errno : 0;
    close(fd);
    return failure;
}

int main(void) {
    @autoreleasepool {
        char command[4096];
        if (!fgets(command, sizeof command, stdin)) return 3;
        int before = 0, granted = 0, siblingDenied = 0;
        NSURL *location = nil;
        if (command[0] == '{') {
            NSData *json = [[NSString stringWithUTF8String:command] dataUsingEncoding:NSUTF8StringEncoding];
            NSDictionary *record = [NSJSONSerialization JSONObjectWithData:json options:0 error:NULL];
            int fd = open([record[@"path"] fileSystemRepresentation], O_RDONLY);
            before = fd < 0 ? errno : 0;
            if (fd >= 0) close(fd);
            NSData *bookmark = [[NSData alloc] initWithBase64EncodedString:record[@"bookmark"] options:0];
            location = [NSURL URLByResolvingBookmarkData:bookmark options:NSURLBookmarkResolutionWithoutUI
                relativeToURL:nil bookmarkDataIsStale:NULL error:NULL];
            NSData *data = location ? [NSData dataWithContentsOfURL:location] : nil;
            granted = [data isEqualToData:[@"fixture" dataUsingEncoding:NSUTF8StringEncoding]];
            fd = open([record[@"sibling"] fileSystemRepresentation], O_RDONLY);
            siblingDenied = fd < 0 ? errno : 0;
            if (fd >= 0) close(fd);
        }
        int tcp = denial(SOCK_STREAM), udp = denial(SOCK_DGRAM);
        pid_t descendant = 0;
        if (!strcmp(command, "spawn\n") || !strcmp(command, "spawnexit\n")) {
            descendant = fork();
            if (descendant < 0) return 4;
            if (descendant == 0) {
                signal(SIGTERM, SIG_IGN);
                for (;;) pause();
            }
        }
        printf("{\"pid\":%d,\"descendant\":%d,\"tcp\":%d,\"udp\":%d,\"heic\":%d,\"similarity\":%d,\"before\":%d,\"granted\":%d,\"sibling\":%d}\n",
            getpid(), descendant, tcp, udp, getenv("PICFETCH_HEIC_WORKER") != NULL,
            getenv("PICFETCH_SIMILARITY_WORKER") != NULL, before, granted, siblingDenied);
        fflush(stdout);
        if (!strcmp(command, "hold\n") || !strcmp(command, "spawn\n")) {
            signal(SIGTERM, SIG_IGN);
            for (;;) pause();
        }
        if (location) [location stopAccessingSecurityScopedResource];
        return 0;
    }
}
