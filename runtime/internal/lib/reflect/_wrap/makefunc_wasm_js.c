#if __SIZEOF_POINTER__ == 4
#include "../../../clite/ffi/wasm32/include/ffi.h"
#else
#include "../../../clite/ffi/wasm64/include/ffi.h"
#endif

extern int llgo_emscripten_asyncify_state(void);
extern void *llgo_reflect_invoke_js(void **args, void *userdata);
extern void llgo_reflect_store1_js(ffi_cif *cif, void *ret, void *userdata,
                                   void *result);
extern void llgo_reflect_storen_js(ffi_cif *cif, void *ret, void *userdata,
                                   void *result);

/* libffi's JavaScript closure trampoline allocates temporary ret and args
 * buffers on every entry, including its replay entry. These functions stay
 * outside Asyncify so a replay calls them with the current buffers. Only the
 * Go invocation below is resumable. */
void llgo_reflect_bind0_js(ffi_cif *cif, void *ret, void **args,
                           void *userdata) {
  (void)cif;
  (void)ret;
  (void)llgo_reflect_invoke_js(args, userdata);
}

void llgo_reflect_bind1_js(ffi_cif *cif, void *ret, void **args,
                           void *userdata) {
  void *result;
  result = llgo_reflect_invoke_js(args, userdata);
  if (llgo_emscripten_asyncify_state() == 0)
    llgo_reflect_store1_js(cif, ret, userdata, result);
}

void llgo_reflect_bindn_js(ffi_cif *cif, void *ret, void **args,
                           void *userdata) {
  void *result = llgo_reflect_invoke_js(args, userdata);
  if (llgo_emscripten_asyncify_state() == 0)
    llgo_reflect_storen_js(cif, ret, userdata, result);
}
