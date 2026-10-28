#ifndef LLGO_PUREGO_FRAME_H
#define LLGO_PUREGO_FRAME_H

#include <stddef.h>
#include <stdint.h>

/* Explicit purego v0.9.1 Linux/amd64 syscall15Args layout. This is neither a
 * Go ABI0 frame nor a generic runtime.cgocall implementation. The adapter
 * marshals spill slots into integer[6:15] in source-argument order. */
typedef struct {
    uintptr_t function;
    uintptr_t integer[15];
    uint64_t floating[8];
    uintptr_t reserved;
} llgo_purego_frame;

_Static_assert(sizeof(uintptr_t) == 8, "purego SysV bridge requires amd64");
_Static_assert(offsetof(llgo_purego_frame, integer) == 8, "integer offset");
_Static_assert(offsetof(llgo_purego_frame, floating) == 128, "float offset");
_Static_assert(offsetof(llgo_purego_frame, reserved) == 192, "reserved offset");
_Static_assert(sizeof(llgo_purego_frame) == 200, "packed frame size");

/* Synchronous C ABI entry. Results replace integer[0:2] and floating[0:2].
 * The caller owns and roots the frame and every pointee until return. */
int32_t llgo_purego_sysv_call(llgo_purego_frame *frame);

#endif
