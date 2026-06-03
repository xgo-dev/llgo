// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.26

package constanttime

import _ "unsafe"

//go:linkname boolToUint8 llgo.boolToUint8
func boolToUint8(b bool) uint8
