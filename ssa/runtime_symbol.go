// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package ssa

import "fmt"

// RuntimeSymbol bounds a computed compiler lookup to an explicit set of names.
// The runtime ownership audit checks every candidate, including paths that the
// current compilation does not execute. Keep candidates literal at call sites.
func RuntimeSymbol(name string, candidates ...string) string {
	for _, candidate := range candidates {
		if name == candidate {
			return name
		}
	}
	panic(fmt.Errorf("runtime selector returned %q outside its audited candidates %q", name, candidates))
}
