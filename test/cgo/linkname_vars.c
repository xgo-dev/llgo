//go:build llgo && !wasm

#include "linkname_vars.h"

static int32_t twice(int32_t value) { return 2 * value; }

int32_t llgo_linkvar_scalar = 42;
int32_t llgo_linkvar_array[3] = {3, 5, 7};
int32_t *llgo_linkvar_pointer = &llgo_linkvar_scalar;
int32_t (*llgo_linkvar_callback)(int32_t) = twice;
int32_t (*llgo_linkvar_callbacks[2])(int32_t) = {twice, twice};
struct llgo_linkvar_callback_box llgo_linkvar_callback_box = {twice, 0xfeedface};

void llgo_linkvar_reset_callbacks(void) {
    llgo_linkvar_callbacks[0] = twice;
    llgo_linkvar_callbacks[1] = twice;
}

int32_t llgo_linkvar_call(int32_t value) {
    return llgo_linkvar_callback(value);
}

int llgo_linkvar_addresses(void *scalar, void *array, void *pointer, void *callback) {
    return scalar == &llgo_linkvar_scalar &&
        array == &llgo_linkvar_array &&
        pointer == &llgo_linkvar_pointer &&
        callback == &llgo_linkvar_callback;
}

int32_t llgo_linkvar_read_scalar(void) { return llgo_linkvar_scalar; }
void llgo_linkvar_write_array(int32_t value) { llgo_linkvar_array[1] = value; }
