// Cortex-M startup and semihosted stdio for QEMU targets.
#include <errno.h>
#include <stdint.h>
#include <sys/types.h>
#include <stdio.h>
#include <time.h>

extern uint32_t _sidata[], _sdata[], _edata[], _sbss[], _ebss[];
extern int main(int argc, char **argv);
extern int SemihostingCall(int operation, const void *argument);

enum {
    SEMIHOST_OPEN = 0x01,
    SEMIHOST_WRITE_CHAR = 0x03,
    SEMIHOST_WRITE_STRING = 0x04,
    SEMIHOST_WRITE = 0x05,
    SEMIHOST_READ_CHAR = 0x07,
    SEMIHOST_CLOCK = 0x10,
    SEMIHOST_TIME = 0x11,
    SEMIHOST_EXIT_EXTENDED = 0x20,
    SEMIHOST_OPEN_WB = 5,
    SEMIHOST_OPEN_AB = 9,
    APPLICATION_EXIT = 0x20026,
};

static int output_handles[] = {-1, -1};

static int put_char(char value, FILE *stream) {
    (void)stream;
    SemihostingCall(SEMIHOST_WRITE_CHAR, &value);
    return 0;
}

static int get_char(FILE *stream) {
    (void)stream;
    return SemihostingCall(SEMIHOST_READ_CHAR, NULL);
}

ssize_t write(int fd, const void *buf, size_t n) {
    if (fd != 1 && fd != 2) {
        errno = EBADF;
        return -1;
    }
    if (n == 0)
        return 0;
    int *handle = &output_handles[fd - 1];
    if (*handle < 0) {
        const uintptr_t open_args[] = {
            (uintptr_t)":tt", fd == 1 ? SEMIHOST_OPEN_WB : SEMIHOST_OPEN_AB, 3
        };
        *handle = SemihostingCall(SEMIHOST_OPEN, open_args);
    }
    if (*handle >= 0) {
        const uintptr_t write_args[] = {(uintptr_t)*handle, (uintptr_t)buf, n};
        int remaining = SemihostingCall(SEMIHOST_WRITE, write_args);
        if (remaining >= 0 && (size_t)remaining <= n)
            return (ssize_t)(n - (size_t)remaining);
    }
    errno = EIO;
    return -1;
}

ssize_t read(int fd, void *buf, size_t n) {
    if (fd != 0) {
        errno = EBADF;
        return -1;
    }
    if (n == 0)
        return 0;
    int value = get_char(NULL);
    if (value < 0)
        return 0;
    *(char *)buf = value;
    return 1;
}

static FILE console = FDEV_SETUP_STREAM(put_char, get_char, NULL, _FDEV_SETUP_RW);
FILE *const stdin = &console;
FILE *const stdout = &console;
FILE *const stderr = &console;

_Noreturn void _exit(int status) {
    uint32_t args[] = {APPLICATION_EXIT, (uint32_t)status};
    SemihostingCall(SEMIHOST_EXIT_EXTENDED, args);
    for (;;) {
        __asm__ volatile("wfe");
    }
}

_Noreturn void exit(int status) {
    _exit(status);
}

time_t time(time_t *result) {
    time_t now = SemihostingCall(SEMIHOST_TIME, NULL);
    if (result)
        *result = now;
    return now;
}

char *getenv(const char *name) {
    (void)name;
    return NULL;
}

int usleep(unsigned usec) {
    if (usec == 0)
        return 0;
    // The semihost clock is measured in centiseconds.
    uint32_t ticks = usec / 10000 + (usec % 10000 != 0);
    int start = SemihostingCall(SEMIHOST_CLOCK, NULL);
    int now = start;
    while (now != -1) {
        if ((uint32_t)now - (uint32_t)start >= ticks)
            return 0;
        now = SemihostingCall(SEMIHOST_CLOCK, NULL);
    }
    errno = EIO;
    return -1;
}

// Programs can provide their own Cortex-M handlers.
__attribute__((weak)) _Noreturn void handleHardFault(uint32_t *frame) {
    (void)frame;
    SemihostingCall(SEMIHOST_WRITE_STRING, "Cortex-M hard fault\n");
    _exit(1);
}

__attribute__((weak)) _Noreturn void Reset_Handler(void) {
    const uint32_t *src = _sidata;
    // Avoid calls to libc before the static storage it uses is ready.
    for (volatile uint32_t *dst = _sdata; dst < _edata; ++dst)
        *dst = *src++;
    for (volatile uint32_t *dst = _sbss; dst < _ebss; ++dst)
        *dst = 0;
    _exit(main(0, NULL));
}
