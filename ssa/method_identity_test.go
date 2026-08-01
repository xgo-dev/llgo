//go:build !llgo
// +build !llgo

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

package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/llgo/ssa/abi"
)

func TestMethodSymbolNamePreservesUnexportedPackageIdentity(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/b", "b")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	aHidden := types.NewFunc(token.NoPos, pkgA, "m", sig)
	bHidden := types.NewFunc(token.NoPos, pkgB, "m", sig)
	aExported := types.NewFunc(token.NoPos, pkgA, "M", sig)

	for _, tt := range []struct {
		method *types.Func
		name   string
		want   string
	}{
		{aHidden, "m", "example.com/a.m"},
		{bHidden, "m", "m"},
		{aExported, "M", "M"},
		{nil, "m", "m"},
	} {
		if got := MethodSymbolName(pkgB, tt.method, tt.name); got != tt.want {
			t.Errorf("MethodSymbolName(%v, %q) = %q, want %q", tt.method, tt.name, got, tt.want)
		}
	}
}

func TestMethodSymbolNameNormalizesPatchedPackageIdentity(t *testing.T) {
	original := types.NewPackage("runtime", "runtime")
	patched := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	hidden := types.NewFunc(token.NoPos, original, "m", sig)

	if got := MethodSymbolName(patched, hidden, hidden.Name()); got != "m" {
		t.Fatalf("MethodSymbolName(patched runtime, runtime.m) = %q, want %q", got, "m")
	}
}

func TestIMethodOfUsesUnexportedPackageIdentity(t *testing.T) {
	pkgA := types.NewPackage("example.com/a", "a")
	pkgB := types.NewPackage("example.com/b", "b")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	aMethod := types.NewFunc(token.NoPos, pkgA, "m", sig)
	bMethod := types.NewFunc(token.NoPos, pkgB, "m", sig)
	intf := types.NewInterfaceType([]*types.Func{bMethod, aMethod}, nil).Complete()

	if got := iMethodOf(intf, aMethod); got < 0 || intf.Method(got) != aMethod {
		t.Fatalf("iMethodOf(a.m) = %d, want a.m slot", got)
	}
	if got := iMethodOf(intf, bMethod); got < 0 || intf.Method(got) != bMethod {
		t.Fatalf("iMethodOf(b.m) = %d, want b.m slot", got)
	}
}
