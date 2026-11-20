/* RISC-V (RV32, ilp32) libffi wrap for baremetal targets.
 *
 * The vendored archive in ../riscv32 provides ffi_prep_cif, ffi_call_go and
 * the ffi_type table. Closures live in ordinary SRAM allocations: the
 * instruction region is executable by construction, so there is no need for
 * an mmap-backed executable page (see llgo_ffi_closure_alloc).
 */
#include "../riscv32/include/ffi.h"

#include <stdlib.h>

_Static_assert(FFI_DEFAULT_ABI == FFI_SYSV,
               "LLGo's riscv32 DefaultAbi constant is stale");
/* Go's ffi.Cif Extra field overlays riscv_nfixedargs (written by
 * ffi_prep_cif_machdep); riscv_unused is never touched. */
_Static_assert(sizeof(ffi_cif) == 8 * sizeof(unsigned),
               "LLGo ffi.Cif does not match riscv32 ffi_cif");

void *llgo_ffi_closure_alloc(void **code) {
  /* Baremetal SRAM (IRAM region) is executable by construction, so a plain
   * allocation doubles as closure code storage. ffi_closure_alloc's mmap
   * path is unavailable, so route through malloc (wrapped to the tinygogc
   * heap on baremetal builds). */
  void *mem = malloc(sizeof(ffi_closure));
  if (mem == 0) {
    *code = 0;
    return 0;
  }
  *code = mem;
  return mem;
}

void llgo_ffi_closure_free(void *closure) { free(closure); }

/* LLVM and libffi both lower the RISC-V static chain through t2 (x7), so
 * ffi_call_go passes env in that register. The CIF therefore never contains
 * env for the nest ABI (see internal/ffi/call_llgo_nest.go). */
void llgo_ffi_call_with_env(ffi_cif *cif, void (*fn)(void), void *rvalue,
                            void **avalue, void *env) {
  ffi_call_go(cif, fn, rvalue, avalue, env);
}

/* Standard closure entry points (also used by the reflect MakeFunc bridge):
 * same ordinary-SRAM allocation as llgo_ffi_closure_alloc. */
void *ffi_closure_alloc(size_t size, void **code) {
  void *mem = malloc(size);
  if (mem == 0) {
    *code = 0;
    return 0;
  }
  *code = mem;
  return mem;
}

void ffi_closure_free(void *closure) { free(closure); }
