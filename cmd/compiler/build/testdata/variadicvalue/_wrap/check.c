#include <stdarg.h>
#include <stdint.h>

int32_t review_check(int32_t receiver, ...) {
    va_list args;
    va_start(args, receiver);
    int a = va_arg(args, int);
    int b = va_arg(args, int);
    va_end(args);
    return receiver == 7 && a == 100 && b == 200 ? 42 : -1;
}
