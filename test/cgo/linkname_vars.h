#ifndef LLGO_TEST_LINKNAME_VARS_H
#define LLGO_TEST_LINKNAME_VARS_H

#include <stdint.h>

extern int32_t llgo_linkvar_scalar;
extern int32_t llgo_linkvar_array[3];
extern int32_t *llgo_linkvar_pointer;
extern int32_t (*llgo_linkvar_callback)(int32_t);
extern int32_t (*llgo_linkvar_callbacks[2])(int32_t);
extern struct llgo_linkvar_callback_box {
    int32_t (*callback)(int32_t);
    uintptr_t guard;
} llgo_linkvar_callback_box;

int32_t llgo_linkvar_call(int32_t value);
int llgo_linkvar_addresses(void *scalar, void *array, void *pointer, void *callback);
int32_t llgo_linkvar_read_scalar(void);
void llgo_linkvar_write_array(int32_t value);
void llgo_linkvar_reset_callbacks(void);

#endif
