#ifndef LLGO_TRACEBACK_H
#define LLGO_TRACEBACK_H
#include <stdint.h>
#include <stddef.h>

/* Keep this in sync with MaxFrames in stacktrace.go. */
#define LLGO_TRACEBACK_MAX (1u << 16)
typedef struct llgo_traceback_snapshot {
    struct llgo_traceback_snapshot *next;
    uint64_t id, parent;
    uintptr_t created;
    uintptr_t *pcs;
    uintptr_t count;
    uint32_t state;
} llgo_traceback_snapshot;

/* A platform-local unwinder, installed and warmed before user code runs. */
typedef size_t (*llgo_traceback_unwinder)(void *, uintptr_t *, size_t, uintptr_t *);
typedef void (*llgo_traceback_thread_init)(void);
void llgo_traceback_set_unwinder(llgo_traceback_unwinder,
                                llgo_traceback_thread_init);
void *llgo_traceback_register(uint64_t, uint64_t, uintptr_t);
void llgo_traceback_attach(void *);
void llgo_traceback_unregister(void *);
void llgo_traceback_state(void *, uint32_t);
llgo_traceback_snapshot *llgo_traceback_capture(uint64_t);
void llgo_traceback_free(llgo_traceback_snapshot *);
uintptr_t *llgo_traceback_fault_buffer(void);
void llgo_traceback_release_fault_buffer(void);

llgo_traceback_snapshot *llgo_traceback_next(llgo_traceback_snapshot *);
uintptr_t *llgo_traceback_info(llgo_traceback_snapshot *, uint64_t *,
                               uint64_t *, uintptr_t *, uintptr_t *, uint32_t *);
#endif
