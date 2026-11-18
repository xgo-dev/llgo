// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package runtime

import _ "unsafe"

// Metadata-only view for cmd/go. LLGo replaces this file with the active
// GOROOT runtime/complex.go; see doc/runtime-source-reuse.md.

//go:linkname complex128div
func complex128div(n complex128, m complex128) complex128
