#include <pthread.h>
#include <stdint.h>

typedef struct {
    void *code;
    uint64_t input;
    uint64_t depth;
    uint64_t result;
    uint64_t callbacks;
} callback_frame;

extern void crosscall2(void *fn, void *frame, int32_t size, uintptr_t context);

static int32_t callback_entry(void *raw)
{
    callback_frame *frame = raw;
    crosscall2(frame->code, frame, 0, 0);
    return -19;
}

static void *foreign_start(void *raw)
{
    callback_entry(raw);
    return 0;
}

static int32_t foreign_entry(void *raw)
{
    pthread_t thread;
    if (pthread_create(&thread, 0, foreign_start, raw) != 0) {
        return -20;
    }
    if (pthread_join(thread, 0) != 0) {
        return -21;
    }
    return -19;
}

void *callback_entry_address(void)
{
    return (void *)&callback_entry;
}

void *foreign_entry_address(void)
{
    return (void *)&foreign_entry;
}
