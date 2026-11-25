/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#if defined(__APPLE__) && defined(__aarch64__)

#if defined(__arm64e__)
#error "llgo_setjmp/llgo_longjmp does not support arm64e pointer authentication"
#endif

/*
 * Fast ABI-only context save/restore for Go defer/panic unwinding on Darwin
 * ARM64. Go defers never save or restore signal masks (fault handlers install
 * with SA_NODEFER and sigsetjmp is always called with savemask=0). Preserving
 * the AAPCS64 callee-saved registers directly avoids the dynamic-linker stub,
 * sigsetjmp wrapper, and pointer-mangling sequence on every deferred frame.
 *
 * Security note: unlike Darwin's libc sigsetjmp, raw fp/lr/sp storage omits
 * per-process pointer-mangling cookie hardening (matching LLGo's existing
 * windows/arm64 llgo_setjmp implementation).
 *
 * Layout fits inside Darwin's 196-byte SigjmpBuf (176 bytes used, 16-byte
 * aligned pairs when env is 16-byte aligned).
 */
__attribute__((naked, noinline, returns_twice)) int llgo_setjmp(void *env) {
    __asm__ volatile(
        "stp x19, x20, [x0, #0]\n\t"
        "stp x21, x22, [x0, #16]\n\t"
        "stp x23, x24, [x0, #32]\n\t"
        "stp x25, x26, [x0, #48]\n\t"
        "stp x27, x28, [x0, #64]\n\t"
        "stp x29, x30, [x0, #80]\n\t"
        "mov x2, sp\n\t"
        "str x2, [x0, #96]\n\t"
        "stp d8, d9, [x0, #112]\n\t"
        "stp d10, d11, [x0, #128]\n\t"
        "stp d12, d13, [x0, #144]\n\t"
        "stp d14, d15, [x0, #160]\n\t"
        "mov w0, wzr\n\t"
        "ret");
}

__attribute__((naked, noinline, noreturn)) void llgo_longjmp(void *env,
                                                             int value) {
    __asm__ volatile(
        "ldp x19, x20, [x0, #0]\n\t"
        "ldp x21, x22, [x0, #16]\n\t"
        "ldp x23, x24, [x0, #32]\n\t"
        "ldp x25, x26, [x0, #48]\n\t"
        "ldp x27, x28, [x0, #64]\n\t"
        "ldp x29, x30, [x0, #80]\n\t"
        "ldr x2, [x0, #96]\n\t"
        "mov sp, x2\n\t"
        "ldp d8, d9, [x0, #112]\n\t"
        "ldp d10, d11, [x0, #128]\n\t"
        "ldp d12, d13, [x0, #144]\n\t"
        "ldp d14, d15, [x0, #160]\n\t"
        "cmp w1, #0\n\t"
        "csinc w0, w1, wzr, ne\n\t"
        "ret");
}

#endif
