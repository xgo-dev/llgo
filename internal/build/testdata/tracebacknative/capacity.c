/* Exercise the real fault capture boundary without creating a huge C stack. */
#include "../../../../runtime/internal/lib/runtime/_wrap/dynunwind.c"
#include "traceback.h"
#include <assert.h>

static uintptr_t storage[LLGO_TRACEBACK_MAX + 1];
static unsigned steps;

uintptr_t *llgo_traceback_fault_buffer(void) { return storage; }
void llgo_traceback_set_unwinder(llgo_traceback_unwinder walk,
                                llgo_traceback_thread_init init)
{
    (void)walk; (void)init;
}

static int fake_init(void *cursor, void *ctx)
{
    (void)cursor; (void)ctx;
    return 0;
}

static int fake_reg(void *cursor, int reg, uintptr_t *value)
{
    (void)cursor;
    *value = reg == dynunw_reg_ip ? 0x1234 : 0;
    return 0;
}

static int fake_step(void *cursor)
{
    (void)cursor;
    return ++steps <= LLGO_TRACEBACK_MAX;
}

int main(void)
{
    storage[LLGO_TRACEBACK_MAX] = 0x5678;
    dynunw_enabled = dynunw_ctx_is_ucontext = 1;
    dynunw_reg_ip = 1;
    dynunw_reg_fp = 2;
    p_init_local = fake_init;
    p_get_reg = fake_reg;
    p_step = fake_step;
    llgo_dynunwind_capture(0);
    assert(storage[LLGO_TRACEBACK_MAX] == 0x5678);
    assert(llgo_dynunwind_pccount() == LLGO_TRACEBACK_MAX);
    assert(storage[LLGO_TRACEBACK_MAX - 1] == 0x1234);
    return 0;
}
