#include "frame.h"

#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

extern int llgo_test_preserved(llgo_purego_frame *frame);
extern uintptr_t llgo_test_alignment(void);
extern uintptr_t llgo_test_vector_count(void);

static void require(int condition, const char *what)
{
    if (!condition) {
        fprintf(stderr, "purego SysV oracle: %s\n", what);
        exit(1);
    }
}

static void invoke(llgo_purego_frame *frame)
{
    require(llgo_test_preserved(frame) == 0, "callee-saved register or caller stack corruption");
}

static uint64_t bits(double value)
{
    uint64_t result;
    memcpy(&result, &value, sizeof(result));
    return result;
}

typedef struct { uint64_t first, second; } integer_pair;
typedef struct { double first, second; } float_pair;

static integer_pair integers(uint64_t a1, uint64_t a2, uint64_t a3, uint64_t a4,
    uint64_t a5, uint64_t a6, uint64_t a7, uint64_t a8, uint64_t a9, uint64_t a10,
    uint64_t a11, uint64_t a12, uint64_t a13, uint64_t a14, uint64_t a15)
{
    uint64_t args[] = {a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15};
    integer_pair result = {0, UINT64_C(0xFEDCBA9876543210)};
    for (unsigned i = 0; i < 15; ++i) {
        result.first += args[i] * (i + 1);
        result.second ^= args[i] << i;
    }
    return result;
}

static float_pair floats(double a1, double a2, double a3, double a4,
    double a5, double a6, double a7, double a8)
{
    return (float_pair){a1 + 2*a2 + 3*a3 + 4*a4 + 5*a5 + 6*a6 + 7*a7 + 8*a8,
        a1 - a2 + a3 - a4 + a5 - a6 + a7 - a8};
}

static double mixed(uint64_t a1, double f1, uint64_t a2, double f2,
    uint64_t a3, double f3, uint64_t a4, double f4, uint64_t a5, double f5,
    uint64_t a6, double f6, uint64_t a7, double f7, uint64_t a8, double f8,
    uint64_t a9, double f9, uint64_t a10, double f10)
{
    return a1 + 2*a2 + 3*a3 + 4*a4 + 5*a5 + 6*a6 + 7*a7 + 8*a8 + 9*a9 + 10*a10 +
        11*f1 + 12*f2 + 13*f3 + 14*f4 + 15*f5 + 16*f6 + 17*f7 + 18*f8 + 19*f9 + 20*f10;
}

static double variadic(unsigned count, ...)
{
    va_list args;
    va_start(args, count);
    double result = 0;
    for (unsigned i = 0; i < count; ++i)
        result += va_arg(args, double) * (i + 1);
    va_end(args);
    return result;
}

static float float32_result(float a, float b)
{
    return a * 3 + b;
}

static uintptr_t nested(const char *value)
{
    llgo_purego_frame inner = {.function = (uintptr_t)&strlen};
    inner.integer[0] = (uintptr_t)value;
    invoke(&inner);
    return inner.integer[0] + 100;
}

int main(void)
{
    llgo_purego_frame frame = {.function = (uintptr_t)&llgo_test_alignment};
    invoke(&frame);
    require(frame.integer[0] == 0, "16-byte call stack alignment");

    frame = (llgo_purego_frame){.function = (uintptr_t)&llgo_test_vector_count};
    invoke(&frame);
    require(frame.integer[0] == 8, "variadic vector-register upper bound");

    frame = (llgo_purego_frame){.function = (uintptr_t)&strlen};
    frame.integer[0] = (uintptr_t)"llgo purego strlen";
    invoke(&frame);
    require(frame.integer[0] == strlen("llgo purego strlen"), "strlen");

    frame = (llgo_purego_frame){.function = (uintptr_t)&integers};
    for (unsigned i = 0; i < 15; ++i)
        frame.integer[i] = i + 11;
    integer_pair expected = integers(11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25);
    invoke(&frame);
    require(frame.integer[0] == expected.first && frame.integer[1] == expected.second,
        "15 integer arguments and RAX/RDX results");

    frame = (llgo_purego_frame){.function = (uintptr_t)&floats};
    for (unsigned i = 0; i < 8; ++i)
        frame.floating[i] = bits(i + 0.25);
    float_pair fp = floats(0.25, 1.25, 2.25, 3.25, 4.25, 5.25, 6.25, 7.25);
    invoke(&frame);
    require(frame.floating[0] == bits(fp.first) && frame.floating[1] == bits(fp.second),
        "8 vector arguments and XMM0/XMM1 results");

    frame = (llgo_purego_frame){.function = (uintptr_t)&mixed};
    for (unsigned i = 0; i < 8; ++i)
        frame.floating[i] = bits(i + 0.5);
    for (unsigned i = 0; i < 9; ++i)
        frame.integer[i] = i + 1;
    frame.integer[9] = bits(8.5);
    frame.integer[10] = 10;
    frame.integer[11] = bits(9.5);
    double mix = mixed(1, 0.5, 2, 1.5, 3, 2.5, 4, 3.5, 5, 4.5, 6, 5.5,
        7, 6.5, 8, 7.5, 9, 8.5, 10, 9.5);
    invoke(&frame);
    require(frame.floating[0] == bits(mix), "interleaved integer/float stack spills");

    frame = (llgo_purego_frame){.function = (uintptr_t)&variadic};
    frame.integer[0] = 10;
    for (unsigned i = 0; i < 8; ++i)
        frame.floating[i] = bits(i + 0.75);
    frame.integer[6] = bits(8.75);
    frame.integer[7] = bits(9.75);
    double va = variadic(10, 0.75, 1.75, 2.75, 3.75, 4.75, 5.75, 6.75, 7.75, 8.75, 9.75);
    invoke(&frame);
    require(frame.floating[0] == bits(va), "variadic float register and spill arguments");

    frame = (llgo_purego_frame){.function = (uintptr_t)&float32_result};
    float f1 = 1.25f, f2 = -0.5f, fexpected = float32_result(f1, f2);
    memcpy(&frame.floating[0], &f1, sizeof(f1));
    memcpy(&frame.floating[1], &f2, sizeof(f2));
    invoke(&frame);
    uint32_t fbits;
    memcpy(&fbits, &fexpected, sizeof(fbits));
    require((uint32_t)frame.floating[0] == fbits, "float32 argument and result payload");

    frame = (llgo_purego_frame){.function = (uintptr_t)&nested};
    frame.integer[0] = (uintptr_t)"nested bridge";
    invoke(&frame);
    require(frame.integer[0] == 113, "nested native call preserves outer frame");
    puts("purego SysV C ABI oracle passed");
    return 0;
}
