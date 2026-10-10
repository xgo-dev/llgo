#include <stdarg.h>

typedef struct {
    long long s8;
    unsigned long long u8;
    long long s16;
    unsigned long long u16;
} narrow_values;

// This aggregate result also exercises the inserted sret parameter.
__attribute__((noinline)) narrow_values narrow_promote(signed char a, unsigned char b, short c, unsigned short d) {
    narrow_values result = {a, b, c, d};
    return result;
}

typedef narrow_values (*narrow_function)(signed char, unsigned char, short, unsigned short);
narrow_function narrow_address(void) { return narrow_promote; }

// The narrow values spill past the argument registers, even with hidden sret.
__attribute__((noinline)) narrow_values narrow_stack(
    int p0, int p1, int p2, int p3, int p4, int p5, int p6, int p7,
    signed char a, unsigned char b, short c, unsigned short d, _Bool flag) {
    narrow_values result = {flag ? a : -a, b, c, d};
    if (p0 + p1 + p2 + p3 + p4 + p5 + p6 + p7 != 36) result.s8 = 0;
    return result;
}
typedef narrow_values (*narrow_stack_function)(
    int, int, int, int, int, int, int, int,
    signed char, unsigned char, short, unsigned short, _Bool);
narrow_stack_function narrow_stack_address(void) { return narrow_stack; }

typedef int (*narrow_arguments_callback)(signed char, unsigned char, short, unsigned short, _Bool);
__attribute__((noinline)) int narrow_invoke(narrow_arguments_callback fn, _Bool flag) {
    return fn(-8, 250, -300, 60000, flag);
}

__attribute__((noinline)) int narrow_variadic(signed char a, short b, _Bool flag, int marker, ...) {
    va_list args;
    va_start(args, marker);
    int tail = va_arg(args, int);
    double number = va_arg(args, double);
    va_end(args);
    return a == -8 && b == -300 && flag && marker == 17 && tail == -128 && number == 1.5 ? 42 : -1;
}
typedef int (*narrow_variadic_function)(signed char, short, _Bool, int, ...);
narrow_variadic_function narrow_variadic_address(void) { return narrow_variadic; }

typedef signed char (*narrow_s8_callback)(int);
typedef unsigned char (*narrow_u8_callback)(int);
typedef short (*narrow_s16_callback)(int);
typedef unsigned short (*narrow_u16_callback)(int);

__attribute__((noinline)) signed char narrow_return_s8(int x) { return (signed char)x; }
__attribute__((noinline)) unsigned char narrow_return_u8(int x) { return (unsigned char)x; }
__attribute__((noinline)) short narrow_return_s16(int x) { return (short)x; }
__attribute__((noinline)) unsigned short narrow_return_u16(int x) { return (unsigned short)x; }

// Volatile hides the callback's identity so LLVM cannot fold the indirect call.
narrow_s8_callback narrow_s8_roundtrip(narrow_s8_callback fn) {
    narrow_s8_callback volatile saved = fn;
    return saved;
}
narrow_u8_callback narrow_u8_roundtrip(narrow_u8_callback fn) {
    narrow_u8_callback volatile saved = fn;
    return saved;
}
narrow_s16_callback narrow_s16_roundtrip(narrow_s16_callback fn) {
    narrow_s16_callback volatile saved = fn;
    return saved;
}
narrow_u16_callback narrow_u16_roundtrip(narrow_u16_callback fn) {
    narrow_u16_callback volatile saved = fn;
    return saved;
}
