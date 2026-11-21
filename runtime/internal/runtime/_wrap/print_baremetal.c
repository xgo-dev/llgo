#include <unistd.h>

// clite uses /dev/stdout for both builtin output streams on this target.
void llgo_baremetal_console_write(const void *data, size_t length) {
    const unsigned char *p = data;
    while (length != 0) {
        ssize_t n = write(STDOUT_FILENO, p, length);
        if (n <= 0)
            return;
        p += n;
        length -= n;
    }
}

void llgo_baremetal_console_byte(unsigned char value) {
    llgo_baremetal_console_write(&value, 1);
}
