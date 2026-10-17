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

package build

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	llclang "github.com/xgo-dev/llgo/internal/clang"
)

func TestClangCVectorTypes(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	const source = `
typedef float F4 __attribute__((vector_size(16)));
typedef F4 Alias;
typedef const F4 Const;
typedef __typeof__(F4) TypeOf;
F4 value;
typedef __typeof__(value) ExprTypeOf;
typedef int I4 __attribute__((ext_vector_type(4)));
typedef unsigned char B16 __attribute__((vector_size(16)));
typedef double D2 __attribute__((vector_size(16)));
typedef float A4[4];
typedef F4 VArray[2];
typedef F4 *Ptr;
struct Pair { F4 first; F4 second; };
`
	got, err := clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), source, nil, map[string]bool{"F4": true, "Alias": true, "Const": true, "TypeOf": true, "ExprTypeOf": true, "I4": true, "B16": true, "D2": true, "A4": true, "VArray": true, "Ptr": true, "struct_Pair": true})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"F4": 4, "Alias": 4, "Const": 4, "TypeOf": 4, "ExprTypeOf": 4, "I4": 4, "B16": 16, "D2": 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, tc := range []struct{ name, source, want string }{
		{"wide", "typedef float F4 __attribute__((vector_size(32)));", "unsupported C vector F4"},
		{"small", "typedef float F4 __attribute__((vector_size(8)));", "unsupported C vector F4"},
		{"unaligned", "typedef float F4 __attribute__((vector_size(16),aligned(1)));", "alignment 1"},
		{"padded lanes", "typedef float F4 __attribute__((ext_vector_type(3)));", "lanes 3"},
		{"anonymous field", "struct Box { float v __attribute__((vector_size(16))); };", "anonymous vector field Box.v"},
		{"anonymous record", "typedef struct { float v __attribute__((vector_size(16))); } F4;", "anonymous vector field F4.v"},
		{"half lanes", "typedef _Float16 F4 __attribute__((vector_size(16)));", "require integer lanes or float/double lanes"},
		{"union", "typedef float V __attribute__((vector_size(16))); typedef union { V v; int i; } F4;", "unsupported C vector record F4"},
		{"packed", "typedef float V __attribute__((vector_size(16))); struct __attribute__((packed)) Box { char tag; V v; };", "unsupported C vector record Box"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), tc.source, nil, map[string]bool{"F4": true, "struct_Box": true})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}
