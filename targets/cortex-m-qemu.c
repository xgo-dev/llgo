// Cortex-M startup and semihosted stdio for the lm3s6965 QEMU target.
#include <errno.h>
#include <stdint.h>
#include <sys/types.h>
#include <stdio.h>
#include <time.h>

extern uint32_t _sidata[], _sdata[], _edata[], _sbss[], _ebss[];
extern int main(int argc, char **argv);
extern int SemihostingCall(int operation, const void *argument);

enum {
    SEMIHOST_WRITE_CHAR = 0x03,
    SEMIHOST_WRITE_STRING = 0x04,
    SEMIHOST_READ_CHAR = 0x07,
    SEMIHOST_TIME = 0x11,
    SEMIHOST_EXIT_EXTENDED = 0x20,
    APPLICATION_EXIT = 0x20026,
};

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
    const char *bytes = buf;
    for (size_t i = 0; i < n; ++i)
        put_char(bytes[i], NULL);
    return (ssize_t)n;
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

_Noreturn void handleHardFault(uint32_t *frame) {
    (void)frame;
    SemihostingCall(SEMIHOST_WRITE_STRING, "Cortex-M hard fault\n");
    _exit(1);
}

_Noreturn void Reset_Handler(void) {
    const uint32_t *src = _sidata;
    // Avoid calls to libc before the static storage it uses is ready.
    for (volatile uint32_t *dst = _sdata; dst < _edata; ++dst)
        *dst = *src++;
    for (volatile uint32_t *dst = _sbss; dst < _ebss; ++dst)
        *dst = 0;
    _exit(main(0, NULL));
}
