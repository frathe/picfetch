// Qualification-only worker: no application data or model is used.
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
    char command[128];
    if (!fgets(command, sizeof command, stdin)) return 3;
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
    printf("{\"pid\":%d,\"descendant\":%d,\"tcp\":%d,\"udp\":%d,\"heic\":%d,\"similarity\":%d}\n",
        getpid(), descendant, tcp, udp, getenv("PICFETCH_HEIC_WORKER") != NULL,
        getenv("PICFETCH_SIMILARITY_WORKER") != NULL);
    fflush(stdout);
    if (!strcmp(command, "hold\n") || !strcmp(command, "spawn\n")) {
        signal(SIGTERM, SIG_IGN);
        for (;;) pause();
    }
    return 0;
}
