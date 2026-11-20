#include <stdint.h>
#include <stddef.h>
#include <string.h>

// The header tests independently check these declarations against LLGo's
// generated header. This fixture executes them against the Wasm C ABI.
typedef struct { void *data; int64_t len, cap; } GoSlice;
extern int64_t llgo_test_signed_word(int64_t n);
extern uint64_t llgo_test_unsigned_word(uint64_t n);
extern GoSlice llgo_test_go_slice(GoSlice s);

static GoSlice make_slice(void *data, int64_t len, int64_t cap) {
    GoSlice s;
    // Give the Memory32 gap between data and len nonzero bytes.
    memset(&s, 0xa5, sizeof(s));
    s.data = data;
    s.len = len;
    s.cap = cap;
    return s;
}

int32_t llgo_test_call_go_exports(void) {
    const int64_t negative = -(INT64_C(1) << 40);
    const uint64_t positive = UINT64_C(1) << 48;
    if (llgo_test_signed_word(negative) != negative - 1) return 1;
    if (llgo_test_unsigned_word(positive) != positive + 1) return 2;
    unsigned char data[5] = {7, 8, 9, 10, 11};
    GoSlice got = llgo_test_go_slice(make_slice(data, 3, 5));
    if (got.data != data + 1 || got.len != 3 || got.cap != 4) return 3;
    return 0;
}

GoSlice llgo_test_c_slice(GoSlice s) {
    unsigned char *data = s.data;
    if (s.len != 3 || s.cap != 5 || data == 0 || data[0] != 7)
        return make_slice(0, 0, 0);
    return make_slice(data + 1, 3, 4);
}

typedef struct { const char *p; int64_t n; } GoString;
typedef struct { GoString Text; GoSlice Data; } Mixed;
extern Mixed llgo_test_go_mixed(Mixed m);

static Mixed make_mixed(const char *text, void *data, int64_t len, int64_t cap) {
    Mixed m;
    memset(&m, 0xa5, sizeof(m));
    m.Text.p = text;
    m.Text.n = 4;
    m.Data.data = data;
    m.Data.len = len;
    m.Data.cap = cap;
    return m;
}

int32_t llgo_test_call_go_mixed(void) {
    unsigned char data[5] = {7, 8, 9, 10, 11};
    Mixed got = llgo_test_go_mixed(make_mixed("wasm", data, 3, 5));
    return got.Text.n == 4 && memcmp(got.Text.p, "wasm", 4) == 0
        && got.Data.data == data + 1 && got.Data.len == 3 && got.Data.cap == 4;
}

Mixed llgo_test_c_mixed(Mixed m) {
    GoSlice s = llgo_test_c_slice(m.Data);
    return make_mixed(m.Text.p, s.data, s.len, s.cap);
}

void llgo_test_call_sink(void (*sink)(GoSlice)) {
    unsigned char data[5] = {7, 8, 9, 10, 11};
    sink(make_slice(data, 3, 5));
}

int32_t llgo_test_call_source(GoSlice (*source)(void)) {
    GoSlice s = source();
    if (s.len != 3 || s.cap < 4 || s.data == 0)
        return -1;
    return *(unsigned char *)s.data;
}
