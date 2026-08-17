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

package layout

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/locality"
)

func TestPlanSharesPointerStorageAndPreservesKinds(t *testing.T) {
	plan, err := Plan("example.com/state", []Declaration{
		{Name: "example.com/state.Ignored", Type: types.Typ[types.Int]},
		{Name: "example.com/state.Z", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Goroutine}},
		{Name: "example.com/state.P", Type: types.NewPointer(types.Typ[types.Int]), Info: locality.Info{Locality: locality.Thread, HasInitializer: true, InitFunc: "p.init1", InitOrder: 2}},
		{Name: "example.com/state.A", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Goroutine, HasInitializer: true, InitFunc: "p.init0", InitOrder: 1}},
		{Name: "example.com/state.Q", Type: types.NewSlice(types.Typ[types.Byte]), Info: locality.Info{Locality: locality.Goroutine}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Variables) != 4 || len(plan.Block) != 2 {
		t.Fatalf("plan sizes = %d/%d", len(plan.Variables), len(plan.Block))
	}
	if got, _ := plan.Lookup("example.com/state.Z"); got.Storage != StorageNativeTLS || got.Info.Locality != locality.Goroutine {
		t.Fatalf("scalar GLS plan = %+v", got)
	}
	if got, _ := plan.Lookup("example.com/state.P"); got.Storage != StoragePackage || got.Field != 0 {
		t.Fatalf("pointer TLS plan = %+v", got)
	}
	if got, _ := plan.Lookup("example.com/state.Q"); got.Storage != StoragePackage || got.Field != 1 {
		t.Fatalf("slice GLS plan = %+v", got)
	}
	if len(plan.Thread) != 1 || plan.Thread[0].Name != "p.init1" {
		t.Fatalf("thread initializers = %+v", plan.Thread)
	}
	if len(plan.Goroutine) != 1 || plan.Goroutine[0].Name != "p.init0" {
		t.Fatalf("goroutine initializers = %+v", plan.Goroutine)
	}
	if got := plan.Initializers(locality.Thread); len(got) != 1 || got[0].Name != "p.init1" {
		t.Fatalf("Initializers(thread) = %+v", got)
	}
	if got := plan.Initializers(locality.Goroutine); len(got) != 1 || got[0].Name != "p.init0" {
		t.Fatalf("Initializers(goroutine) = %+v", got)
	}
	if got := plan.Initializers(locality.None); got != nil {
		t.Fatalf("Initializers(none) = %+v", got)
	}
	if _, ok := plan.Lookup("example.com/state.Missing"); ok {
		t.Fatal("Lookup found a missing variable")
	}
}

func TestOrderedInitializers(t *testing.T) {
	got := orderedInitializers(map[int]string{3: "p.third", 1: "p.first", 2: "p.second"})
	if len(got) != 3 || got[0].Name != "p.first" || got[1].Name != "p.second" || got[2].Name != "p.third" {
		t.Fatalf("ordered initializers = %+v", got)
	}
}

func TestPlanRejectsInvalidDeclarations(t *testing.T) {
	tests := []struct {
		name string
		in   []Declaration
		want string
	}{
		{"missing type", []Declaration{{Name: "p.x", Info: locality.Info{Locality: locality.Thread}}}, "incomplete"},
		{"duplicate", []Declaration{{Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread}}, {Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread}}}, "duplicate"},
		{"invalid kind", []Declaration{{Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Kind(99)}}}, "invalid locality"},
		{"unprepared", []Declaration{{Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread, HasInitializer: true}}}, "inconsistent initializer metadata"},
		{"unexpected helper", []Declaration{{Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread, InitFunc: "p.a", InitOrder: 1}}}, "inconsistent initializer metadata"},
		{"order conflict", []Declaration{{Name: "p.x", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread, HasInitializer: true, InitFunc: "p.a", InitOrder: 1}}, {Name: "p.y", Type: types.Typ[types.Int], Info: locality.Info{Locality: locality.Thread, HasInitializer: true, InitFunc: "p.b", InitOrder: 1}}}, "names both"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Plan("p", test.in); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Plan error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestNames(t *testing.T) {
	if got := BlockName("example.com/p"); got != "example.com/p.__llgo_local_block" {
		t.Fatal(got)
	}
	if got := BlockCacheName("example.com/p"); got != "example.com/p.__llgo_local_cache" {
		t.Fatal(got)
	}
	if got := InitName("example.com/p", locality.Thread); got != "example.com/p.__llgo_tls_init" {
		t.Fatal(got)
	}
	if got := EnsureName("example.com/p", locality.Goroutine); got != "example.com/p.__llgo_gls_init$ensure" {
		t.Fatal(got)
	}
	if got := GuardName("", locality.Thread); got != "__llgo_tls_init$guard" {
		t.Fatal(got)
	}
	if got := FailureCacheName("", locality.Goroutine); got != "__llgo_gls_init$failure_cache" {
		t.Fatal(got)
	}
}

func TestStorageForType(t *testing.T) {
	if got := StorageForType(types.Typ[types.Uintptr]); got != StorageNativeTLS {
		t.Fatalf("uintptr storage = %v", got)
	}
	if got := StorageForType(types.NewArray(types.NewPointer(types.Typ[types.Int]), 0)); got != StorageNativeTLS {
		t.Fatalf("zero-length pointer array storage = %v", got)
	}
	if got := StorageForType(types.NewStruct([]*types.Var{
		types.NewField(0, nil, "value", types.NewPointer(types.Typ[types.Int]), false),
	}, nil)); got != StoragePackage {
		t.Fatalf("pointer struct storage = %v", got)
	}
}

func TestHasPointers(t *testing.T) {
	pkg := types.NewPackage("example.com/types", "types")
	namedInt := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Int", nil), types.Typ[types.Int], nil)
	typeParam := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewInterfaceType(nil, nil).Complete())
	tests := []struct {
		typ  types.Type
		want bool
	}{
		{types.Typ[types.Int], false},
		{types.Typ[types.String], true},
		{types.Typ[types.UnsafePointer], true},
		{types.NewPointer(types.Typ[types.Int]), true},
		{types.NewSlice(types.Typ[types.Int]), true},
		{types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), true},
		{types.NewChan(types.SendRecv, types.Typ[types.Int]), true},
		{types.NewSignatureType(nil, nil, nil, nil, nil, false), true},
		{types.NewInterfaceType(nil, nil).Complete(), true},
		{types.NewArray(types.Typ[types.Int], 1), false},
		{types.NewArray(types.NewPointer(types.Typ[types.Int]), 0), false},
		{types.NewArray(types.NewPointer(types.Typ[types.Int]), 1), true},
		{types.NewStruct([]*types.Var{types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])}, nil), false},
		{types.NewStruct([]*types.Var{types.NewVar(token.NoPos, pkg, "p", types.NewPointer(types.Typ[types.Int]))}, nil), true},
		{namedInt, false},
		{typeParam, true},
		{types.NewTuple(), false},
	}
	for _, test := range tests {
		if got := hasPointers(test.typ); got != test.want {
			t.Fatalf("hasPointers(%v) = %v, want %v", test.typ, got, test.want)
		}
	}
}
