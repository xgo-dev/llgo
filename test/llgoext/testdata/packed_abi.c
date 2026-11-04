// No headers are needed: this fixture is also compiled for foreign targets.
extern int puts(const char *);

typedef struct __attribute__((packed)) { unsigned char tag; long long value; } Packed;
typedef struct { Packed value; } Nested;
typedef struct __attribute__((packed)) { unsigned char tag[1]; long long tail[0]; } Zero1;
typedef struct __attribute__((packed)) { unsigned char tag[2]; long long tail[0]; } Zero2;
typedef struct __attribute__((packed)) { unsigned char tag[8]; long long tail[0]; } Zero8;
typedef struct __attribute__((packed)) { int value; unsigned char tag; } Stride;
typedef struct { Stride values[2]; } Array;
typedef struct __attribute__((aligned(16))) { Packed value; } Aligned;

#define CASE(name, T, equal) \
    T c_##name(T value) { return value; } \
    T native_indirect_##name(T (*fn)(T), T value) { return fn(value); } \
    extern T llgo_direct_##name(T); \
    extern T llgo_indirect_##name(T (*)(T), T); \
    static int equal_##name(T a, T b) { return equal; } \
    static int check_##name(T value) { \
        T (*volatile fn)(T) = c_##name; \
        if (!equal_##name(value, llgo_direct_##name(value)) || \
            !equal_##name(value, llgo_indirect_##name(fn, value))) { \
            puts(#name); \
            return 1; \
        } \
        return 0; \
    }

CASE(packed, Packed, a.tag == b.tag && a.value == b.value)
CASE(nested, Nested, a.value.tag == b.value.tag && a.value.value == b.value.value)
CASE(zero1, Zero1, a.tag[0] == b.tag[0])
CASE(zero2, Zero2, a.tag[0] == b.tag[0] && a.tag[1] == b.tag[1])
CASE(zero8, Zero8,
     a.tag[0] == b.tag[0] && a.tag[1] == b.tag[1] &&
     a.tag[2] == b.tag[2] && a.tag[3] == b.tag[3] &&
     a.tag[4] == b.tag[4] && a.tag[5] == b.tag[5] &&
     a.tag[6] == b.tag[6] && a.tag[7] == b.tag[7])
CASE(array, Array,
     a.values[0].value == b.values[0].value && a.values[0].tag == b.values[0].tag &&
     a.values[1].value == b.values[1].value && a.values[1].tag == b.values[1].tag)
CASE(aligned, Aligned, a.value.tag == b.value.tag && a.value.value == b.value.value)

int main(void) {
    Packed packed = {17, 0x123456789abcdefLL};
    Nested nested = {{29, -0x123456789abcdefLL}};
    Zero1 zero1 = {{37}};
    Zero2 zero2 = {{41, 53}};
    Zero8 zero8 = {{1, 3, 5, 7, 11, 13, 17, 19}};
    Array array = {{{0x12345678, 61}, {-0x12345678, 71}}};
    Aligned aligned = {{83, 0x123456789abcdefLL}};
    return check_packed(packed) || check_nested(nested) ||
           check_zero1(zero1) || check_zero2(zero2) || check_zero8(zero8) ||
           check_array(array) || check_aligned(aligned);
}
