#include "traceback.h"

llgo_traceback_snapshot *llgo_traceback_next(llgo_traceback_snapshot *s)
{
    return s->next;
}

uintptr_t *llgo_traceback_info(llgo_traceback_snapshot *s, uint64_t *id,
                               uint64_t *parent, uintptr_t *created,
                               uintptr_t *count, uint32_t *state)
{
    *id = s->id;
    *parent = s->parent;
    *created = s->created;
    *count = s->count;
    *state = s->state;
    return s->pcs;
}

uintptr_t *llgo_traceback_creator(llgo_traceback_snapshot *s, uintptr_t *count)
{
    *count = s->creator_count;
    return s->creator_pcs;
}
