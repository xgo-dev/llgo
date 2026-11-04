#include <stdint.h>

typedef struct {
    uint64_t left;
    uint64_t right;
    uint64_t result;
    uint64_t calls;
} packed_frame;

static int32_t packed_entry(void *raw)
{
    packed_frame *frame = raw;
    frame->result = frame->left ^ frame->right;
    frame->calls++;
    return -37;
}

void *packed_entry_address(void)
{
    return (void *)&packed_entry;
}
