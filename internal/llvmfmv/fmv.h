#ifndef LLGO_SIMD_FMV_H
#define LLGO_SIMD_FMV_H

#ifdef __cplusplus
extern "C" {
#endif

// Runs the mandatory, early SIMD multiversioning transform. A non-null result
// is a malloc-allocated diagnostic owned by the caller.
char *llgoRunSIMDFMV(void *module);

#ifdef __cplusplus
}
#endif

#endif
