#include <stdint.h>
#include <stddef.h>
#include <string.h>

// The header tests independently check these declarations against LLGo's
// generated header. This fixture executes them against the Wasm C ABI.
typedef struct { void *data; int64_t len, cap; } GoSlice;
typedef struct { const char *p; int64_t n; } GoString;
extern int32_t llgo_test_go_string(GoString s);
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

static void dirty_string_padding(GoString *s) {
    memset((unsigned char *)s + sizeof(s->p), 0xa5, offsetof(GoString, n) - sizeof(s->p));
}

int32_t llgo_test_call_go_exports(void) {
    const int64_t negative = -(INT64_C(1) << 40);
    const uint64_t positive = UINT64_C(1) << 48;
    if (llgo_test_signed_word(negative) != negative - 1) return 1;
    if (llgo_test_unsigned_word(positive) != positive + 1) return 2;
    unsigned char data[5] = {7, 8, 9, 10, 11};
    GoSlice got = llgo_test_go_slice(make_slice(data, 3, 5));
    if (got.data != data + 1 || got.len != 3 || got.cap != 4) return 3;
    GoString text = {"wasm", 4};
    dirty_string_padding(&text);
    if (llgo_test_go_string(text) != 'w') return 4;
    return 0;
}

GoSlice llgo_test_c_slice(GoSlice s) {
    unsigned char *data = s.data;
    if (s.len != 3 || s.cap != 5 || data == 0 || data[0] != 7)
        return make_slice(0, 0, 0);
    return make_slice(data + 1, 3, 4);
}

typedef struct { GoString Text; GoSlice Data; } Mixed;
extern Mixed llgo_test_go_mixed(Mixed m);

static unsigned char host_bytes[5] = {7, 8, 9, 10, 11};
struct { Mixed headers; uint64_t word; Mixed rows[1][2]; } llgo_test_host_global = {
    .headers = {{"wasm", 4}, {host_bytes, 3, 5}}, .word = UINT64_C(0x123456789abcdef0)
};
struct { Mixed headers; uint64_t word; } llgo_test_plain_global;

GoString *llgo_test_dirty_host_headers(void) {
    Mixed *m = &llgo_test_host_global.headers;
    memset((unsigned char *)&m->Text + sizeof(m->Text.p), 0xa5,
           offsetof(GoString, n) - sizeof(m->Text.p));
    memset((unsigned char *)&m->Data + sizeof(m->Data.data), 0xa5,
           offsetof(GoSlice, len) - sizeof(m->Data.data));
    memcpy(&llgo_test_host_global.rows[0][0], m, sizeof(*m));
    memcpy(&llgo_test_host_global.rows[0][1], m, sizeof(*m));
    memcpy(&llgo_test_plain_global.headers, m, sizeof(*m));
    llgo_test_plain_global.word = llgo_test_host_global.word;
    return &llgo_test_host_global.rows[0][1].Text;
}

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

typedef struct { GoString text[4097]; } LargeText;

extern int32_t llgo_test_go_large_param(LargeText);

static LargeText *large_text(void) {
    // Keep the source object above the 64 KiB ABI threshold in static storage.
    static LargeText result;
    for (size_t i = 0; i < sizeof(result.text) / sizeof(result.text[0]); i++) {
        result.text[i] = (GoString){"wasm", 4};
        dirty_string_padding(&result.text[i]);
    }
    return &result;
}

LargeText llgo_test_large_result(void) {
    return *large_text();
}

int32_t llgo_test_call_large_param(void) {
    LargeText *src = large_text();
    const size_t count = sizeof(src->text) / sizeof(src->text[0]);
    const char *samples[] = {"head", "body", "tail"};
    for (size_t i = 0; i < 3; i++) {
        GoString *s = &src->text[i * (count - 1) / 2];
        s->p = samples[i];
        dirty_string_padding(s);
    }
    int32_t status = llgo_test_go_large_param(*src);
    for (size_t i = 0; i < count; i++) {
        const volatile unsigned char *bytes = (const volatile unsigned char *)&src->text[i];
        for (size_t j = sizeof(src->text[i].p); j < offsetof(GoString, n); j++) {
            if (bytes[j] != 0xa5) return -1;
        }
    }
    return status;
}

typedef struct { GoSlice Data[64]; } ProjectionSlices;
typedef struct { GoString Data[64]; } ProjectionStrings;
static ProjectionSlices projection_slices;
static ProjectionStrings projection_strings;

ProjectionSlices llgo_test_projection_slices(void) { return projection_slices; }
ProjectionStrings llgo_test_projection_strings(void) { return projection_strings; }

extern int64_t llgo_test_slice_len(ProjectionSlices);
extern int64_t llgo_test_slice_cap(ProjectionSlices);
extern int64_t llgo_test_string_len(ProjectionStrings);
extern int64_t llgo_test_result_slice_len(void);
extern int64_t llgo_test_result_slice_cap(void);
extern int64_t llgo_test_result_string_len(void);
extern uint64_t llgo_test_projection_slice_slot(ProjectionSlices);
extern uint64_t llgo_test_projection_result_slot(void);

static int unchanged_bytes(const volatile void *object, const void *before, size_t size) {
    const volatile unsigned char *bytes = object;
    const unsigned char *expected = before;
    for (size_t i = 0; i < size; i++) {
        if (bytes[i] != expected[i]) return 0;
    }
    return 1;
}

int32_t llgo_test_call_header_projections(void) {
    static unsigned char data[5] = {7, 8, 9, 10, 11};
    const char *text = "projection";
    memset(&projection_slices, 0xa5, sizeof(projection_slices));
    memset(&projection_strings, 0xa5, sizeof(projection_strings));
    for (size_t i = 0; i < 64; i++) {
        projection_slices.Data[i].data = data;
        projection_slices.Data[i].len = 3;
        projection_slices.Data[i].cap = 5;
        projection_strings.Data[i].p = text;
        projection_strings.Data[i].n = 10;
    }
    projection_slices.Data[63].cap = 4;
    ProjectionSlices slices_before;
    ProjectionStrings strings_before;
    memcpy(&slices_before, &projection_slices, sizeof(slices_before));
    memcpy(&strings_before, &projection_strings, sizeof(strings_before));
    if (llgo_test_slice_len(projection_slices) != 3 ||
        llgo_test_slice_cap(projection_slices) != 4 ||
        llgo_test_string_len(projection_strings) != 10) return 1;
    if (llgo_test_result_slice_len() != 3 || llgo_test_result_slice_cap() != 4 ||
        llgo_test_result_string_len() != 10) return 2;
    if (llgo_test_projection_slice_slot(projection_slices) != (uint64_t)(uintptr_t)data ||
        llgo_test_projection_result_slot() != (uint64_t)(uintptr_t)text) return 3;
    // Read the original bytes even if C assumes a by-value call preserves them.
    if (!unchanged_bytes(&projection_slices, &slices_before, sizeof(slices_before)) ||
        !unchanged_bytes(&projection_strings, &strings_before, sizeof(strings_before))) return 4;
    return 0;
}

int32_t llgo_test_read_string(int32_t (*read)(GoString)) {
    GoString text = {"wasm", 4};
    dirty_string_padding(&text);
    return read(text);
}

typedef GoString (*StringSource)(void);
typedef GoSlice (*SliceSource)(void);

int32_t llgo_test_read_source(StringSource source) {
    if (source == NULL) return -1;
    GoString text = source();
    return text.n > 0 ? text.p[0] : -1;
}

static GoString first_text(void)  { return (GoString){"first", 5}; }
static GoString second_text(void) { return (GoString){"second", 6}; }
static unsigned char source_bytes[2] = {1, 2};
static GoSlice first_data(void)   { return make_slice(source_bytes, 1, 2); }
static GoSlice second_data(void)  { return make_slice(source_bytes + 1, 1, 1); }

StringSource llgo_test_string_source(int32_t second) {
    if (second < 0) return NULL;
    return second ? second_text : first_text;
}

SliceSource llgo_test_slice_source(int32_t second) {
    if (second < 0) return NULL;
    return second ? second_data : first_data;
}

int32_t llgo_test_match_sources(int32_t second, StringSource text, SliceSource data) {
    return text == llgo_test_string_source(second) && data == llgo_test_slice_source(second);
}

typedef struct { int32_t value; } Box;

GoString llgo_test_generic_source(Box arg) {
    return arg.value == 17 ? (GoString){"ok", 2} : (GoString){0};
}

int32_t llgo_test_match_generic(GoString (*f)(Box)) {
    return f == llgo_test_generic_source;
}

typedef struct { struct { const char *data; uint32_t tag; } slot; int64_t len; } TaggedText;

static GoString reflect_text(uintptr_t word) {
    GoString s = {word == 515 ? "word" : "fail", 4};
    if ((uint64_t)word == UINT64_C(0x10000000203)) s.p = "wide";
    dirty_string_padding(&s);
    return s;
}

static TaggedText reflect_tagged(uintptr_t word) {
    return (TaggedText){{"word", word == 515 ? UINT32_C(0x12345678) : 0}, 4};
}

typedef GoString (*ReflectText)(uintptr_t);
typedef TaggedText (*ReflectTagged)(uintptr_t);
ReflectText llgo_test_reflect_text(void) { return reflect_text; }
ReflectTagged llgo_test_reflect_tagged(void) { return reflect_tagged; }
