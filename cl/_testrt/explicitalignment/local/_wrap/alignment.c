#include <stddef.h>
#include <stdint.h>

struct __attribute__((aligned(64))) Record {
    uint8_t tag;
    uint64_t value;
    uint32_t tail;
};

int32_t llgo_explicit_alignment_check(struct Record *records, uintptr_t size,
                                    uintptr_t alignment, uintptr_t value_offset,
                                    uintptr_t tail_offset) {
    if (size != sizeof(struct Record) || alignment != __alignof__(struct Record) ||
        value_offset != offsetof(struct Record, value) ||
        tail_offset != offsetof(struct Record, tail))
        return 1;
    if ((uintptr_t)records % 64 != 0 || (uintptr_t)&records[1] % 64 != 0)
        return 2;
    if (records[0].tag != 1 || records[0].value != 17 || records[0].tail != 91 ||
        records[1].tag != 2 || records[1].value != 29 || records[1].tail != 93)
        return 3;
    records[1].value = 41;
    records[0].tail = 97;
    return 0;
}
