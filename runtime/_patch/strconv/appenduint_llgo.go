// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

//go:build go1.27

// The facade indirection this patch replaces (strconv.AppendUint forwarding
// to internal/strconv) exists from Go 1.27; on older toolchains the skipped
// upstream definitions must stay in place.

package strconv

//llgo:skip AppendUint

import _ "unsafe"

// LLGo patch: the upstream wrapper is one call layer that llgo cannot
// inline (slices cross the frame through a spill slot). Bind the exported
// name directly to the (LLGo-patched) internal implementation symbol; the
// behavior is identical because it is the very same function.
//
//go:linkname AppendUint internal/strconv.AppendUint
func AppendUint(dst []byte, i uint64, base int) []byte
