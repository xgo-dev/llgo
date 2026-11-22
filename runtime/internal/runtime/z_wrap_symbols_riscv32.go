//go:build tinygo.riscv32

// Copyright 2025 The llgo Authors. All rights reserved.
// Use of this source code is governed by an Apache-style license that can be
// found in the LICENSE file.

package runtime

// newlib-esp32 ships RV32 builds of only its PREFER_SIZE_OVER_SPEED memmove,
// memcpy and memset - plain byte loops - and LLVM's loop-idiom pass folds Go
// copy and zero loops into calls to those libc names in every package IR,
// so byte-loop calls appear far beyond the call sites that route to the
// llgo-prefixed helpers explicitly. Wrapping the libc names redirects every
// such reference (Go lowering and newlib's own stdio/realloc internals
// alike) to the word-block implementations in _wrap/memmove_riscv32.c.
//
// The wrap symbols live in this package's archive: programs that link the
// runtime get both the flags and their providers. Runtime-free links see
// neither, and resolve the libc names as before.
const LLGoPackage = "link: --wrap=memcpy --wrap=memmove --wrap=memset"
