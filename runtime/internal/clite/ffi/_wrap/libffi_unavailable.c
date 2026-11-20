#include <stddef.h>

enum { LLGO_FFI_BAD_ABI = 2 };

unsigned int ffi_prep_cif(void *cif, unsigned int abi, unsigned int nargs,
                          void *rtype, void *atypes) {
  return LLGO_FFI_BAD_ABI;
}

unsigned int ffi_prep_cif_var(void *cif, unsigned int abi,
                              unsigned int nfixedargs,
                              unsigned int ntotalargs, void *rtype,
                              void *atypes) {
  return LLGO_FFI_BAD_ABI;
}

void ffi_call(void *cif, void (*fn)(void), void *rvalue, void *avalue) {
  __builtin_trap();
}

void llgo_ffi_call_with_env(void *cif, void (*fn)(void), void *rvalue,
                            void *avalue, void *env) {
  __builtin_trap();
}

void *llgo_ffi_closure_alloc(void **code) {
  *code = NULL;
  return NULL;
}

void ffi_closure_free(void *closure) {}

unsigned int ffi_prep_closure_loc(void *closure, void *cif, void *fn,
                                  void *userdata, void *codeloc) {
  return LLGO_FFI_BAD_ABI;
}
