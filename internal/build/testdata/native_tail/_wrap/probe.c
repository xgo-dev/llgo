#include <stdint.h>

#define GUEST_BODY \
 "sub sp, sp, #32\n" \
 "mov x29, sp\n" \
 "ldr x9, [x0]\n" \
 "add x9, x9, #42\n" \
 "str x9, [x0, #8]\n" \
 "mov x19, #19\n" \
 "mov x20, #20\n" \
 "mov x21, #21\n" \
 "mov x22, #22\n" \
 "mov x23, #23\n" \
 "mov x24, #24\n" \
 "mov x25, #25\n" \
 "mov x26, #26\n" \
 "mov x27, #27\n" \
 "mov x28, #28\n" \
 "fmov d8, x19\n" \
 "fmov d9, x19\n" \
 "fmov d10, x19\n" \
 "fmov d11, x19\n" \
 "fmov d12, x19\n" \
 "fmov d13, x19\n" \
 "fmov d14, x19\n" \
 "fmov d15, x19\n" \
 "ldr x29, [x0, #16]\n" \
 "ldr x30, [x0, #32]\n" \
 "ldr x9, [x0, #24]\n" \
 "mov sp, x9\n" \
 "ret\n"

__attribute__((naked)) static void guest(void) {
 __asm__ volatile(
 "str x29, [x0, #16]\n"
 "mov x9, sp\n"
 "str x9, [x0, #24]\n"
 "str x30, [x0, #32]\n"
 "mov sp, x26\n"
 GUEST_BODY);
}

__attribute__((naked)) static void resume_guest(void) {
 __asm__ volatile(GUEST_BODY);
}

uintptr_t native_tail_guest_addr(void) { return (uintptr_t)&guest; }
uintptr_t native_tail_resume_addr(void) { return (uintptr_t)&resume_guest; }

int native_tail_check(uintptr_t entry, uintptr_t code, uintptr_t context, uintptr_t stack) {
 register uint64_t r19 __asm__("x19") = 1019;
 register uint64_t r20 __asm__("x20") = 1020;
 register uint64_t r21 __asm__("x21") = 1021;
 register uint64_t r22 __asm__("x22") = 1022;
 register uint64_t r23 __asm__("x23") = 1023;
 register uint64_t r24 __asm__("x24") = 1024;
 register uint64_t r25 __asm__("x25") = 1025;
 register uint64_t r26 __asm__("x26") = 1026;
 register uint64_t r27 __asm__("x27") = 1027;
 register uint64_t r28 __asm__("x28") = 1028;
 register double v8 __asm__("d8") = 1008.0;
 register double v9 __asm__("d9") = 1009.0;
 register double v10 __asm__("d10") = 1010.0;
 register double v11 __asm__("d11") = 1011.0;
 register double v12 __asm__("d12") = 1012.0;
 register double v13 __asm__("d13") = 1013.0;
 register double v14 __asm__("d14") = 1014.0;
 register double v15 __asm__("d15") = 1015.0;
 __asm__ volatile("" : "+r"(r19), "+r"(r20), "+r"(r21), "+r"(r22), "+r"(r23), "+r"(r24), "+r"(r25), "+r"(r26), "+r"(r27), "+r"(r28), "+w"(v8), "+w"(v9), "+w"(v10), "+w"(v11), "+w"(v12), "+w"(v13), "+w"(v14), "+w"(v15));
 ((void (*)(uintptr_t, uintptr_t, uintptr_t))entry)(code, context, stack);
 __asm__ volatile("" : "+r"(r19), "+r"(r20), "+r"(r21), "+r"(r22), "+r"(r23), "+r"(r24), "+r"(r25), "+r"(r26), "+r"(r27), "+r"(r28), "+w"(v8), "+w"(v9), "+w"(v10), "+w"(v11), "+w"(v12), "+w"(v13), "+w"(v14), "+w"(v15));
 return r19 == 1019 && r20 == 1020 && r21 == 1021 && r22 == 1022 && r23 == 1023 && r24 == 1024 && r25 == 1025 && r26 == 1026 && r27 == 1027 && r28 == 1028 && v8 == 1008.0 && v9 == 1009.0 && v10 == 1010.0 && v11 == 1011.0 && v12 == 1012.0 && v13 == 1013.0 && v14 == 1014.0 && v15 == 1015.0;
}
