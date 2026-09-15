#include <stddef.h>

size_t llgo_wasm_profile_pointer_size(void) { return sizeof(void *); }

size_t llgo_wasm_profile_long_size(void) { return sizeof(long); }

size_t llgo_wasm_profile_echo_size(size_t value) { return value; }
