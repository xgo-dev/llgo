#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>

static void print_decimal(uint64_t value, bool negative) {
    char buf[21];
    char *cursor = buf + sizeof(buf);
    do {
        *--cursor = '0' + (char)(value % 10);
        value /= 10;
    } while (value);
    if (negative)
        *--cursor = '-';
    fwrite(cursor, 1, (size_t)(buf + sizeof(buf) - cursor), stderr);
}

void llgo_print_int(int64_t value) {
    uint64_t magnitude = (uint64_t)value;
    bool negative = value < 0;
    print_decimal(negative ? 0 - magnitude : magnitude, negative);
}

void llgo_print_uint(uint64_t value) {
    print_decimal(value, false);
}
