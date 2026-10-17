#include <stdarg.h>
#include <stdint.h>

int32_t variadic_method_check(const int32_t *marker, ...) {
    if (*marker == 0) return 42;
    if (*marker != 7) return -1;
    va_list args;
    va_start(args, marker);
    int ok = 1;
    for (int i = 0; i < 10; i++) {
        if (va_arg(args, int) != 100 + i) ok = 0;
    }
    for (int i = 0; i < 10; i++) {
        if (va_arg(args, double) != i + 0.25) ok = 0;
    }
    va_end(args);
    return ok ? 42 : -1;
}
