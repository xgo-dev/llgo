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
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"

	llclang "github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llvm"
)

func TestClangCVectorTypes(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	const source = `
typedef float Scalar;
typedef Scalar F4 __attribute__((vector_size(16)));
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
		{"union vector arrays", "typedef float V __attribute__((vector_size(16))); typedef union { V a[2]; V b[2]; } F4;", "unsupported C vector record F4"},
		{"union array alias", "typedef float V __attribute__((vector_size(16))); typedef V A[2]; typedef union { A a; A b; } F4;", "unsupported C vector record F4"},
		{"local typedef shadow", "typedef float V __attribute__((vector_size(16))); typedef union { V a[2]; } F4; static void f(void) { typedef int V; V local; }", "unsupported C vector record F4"},
		{"union nested record", "typedef float V __attribute__((vector_size(16))); struct Inner { V v; }; typedef union { struct Inner a[2]; int b; } F4;", "unsupported C vector record F4"},
		{"union anonymous record", "typedef float V __attribute__((vector_size(16))); typedef union { struct { V v; } a; int b; } F4;", "unsupported C vector record F4"},
		{"field alignment", "typedef float V __attribute__((vector_size(16))); struct Box { char tag; V v __attribute__((aligned(32))); char tail; };", "unsupported C vector record Box"},
		{"scalar field alignment", "typedef float V __attribute__((vector_size(16))); struct Box { char tag __attribute__((aligned(32))); V v; };", "unsupported C vector record Box"},
		{"nested packed record", "typedef float V __attribute__((vector_size(16))); struct __attribute__((packed)) Inner { V v; }; struct Box { struct Inner a[2]; };", "unsupported C vector record Inner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), tc.source, nil, map[string]bool{"F4": true, "struct_Box": true})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
	// Vector pointers do not impose vector alignment on the record.
	for _, source := range []string{
		"typedef float V __attribute__((vector_size(16))); typedef V *P; struct __attribute__((packed)) Box { P a[2]; };",
		"typedef float V __attribute__((vector_size(16))); typedef struct { V v; } Inner; struct Box { Inner a[2]; };",
	} {
		if _, err := clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), source, nil, map[string]bool{"struct_Box": true}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBuildCVectorTypes(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang unavailable")
	}
	for _, tc := range []struct{ fixture, want string }{
		{"./testdata/cvector", ""},
		{"./testdata/cvector-wide", "unsupported C vector F8"},
		{"./testdata/cvector-map-key", "vector storage is not supported in map key"},
		{"./testdata/cvector-map-value", "vector storage is not supported in map value"},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			pkgs, err := Do([]string{tc.fixture}, NewDefaultConf(ModeGen))
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want %s", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 {
				t.Fatalf("packages = %d, want 1", len(pkgs))
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err, mod.String())
			}
			word := pkgs[0].LPkg.Prog.GoWordSize() * 8
			wantLayout := fmt.Sprintf("[i%d 48, i%d 16, i%d 16, i%d 4]", word, word, word, word)
			if !strings.Contains(mod.String(), wantLayout) {
				t.Fatal("incorrect C source layout", mod.String())
			}
			if !strings.Contains(mod.String(), "<4 x float>") {
				t.Fatal("C vector representation lost", mod.String())
			}
		})
	}
}

func TestClangCVectorDiagnostics(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	_, err = clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), "this is not C", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "inspect Clang types") {
		t.Fatalf("error = %v", err)
	}
	got, err := clangCVectorTypes(llclang.NewCompiler(llclang.Config{CC: clang}), "typedef float A4[4];", nil, map[string]bool{"A4": true})
	if err != nil || len(got) != 0 {
		t.Fatalf("ordinary array = %v, %v", got, err)
	}
}

func TestCVectorUnalignedLLVM(t *testing.T) {
	conf := NewDefaultConf(ModeGen)
	pkgs, err := Do([]string{"./testdata/cvector-unaligned"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	mod.SetTarget(pkg.Prog.Target().Spec().Triple)
	mod.SetDataLayout(pkg.Prog.DataLayout())
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, op, vector, alignment string
	}{
		{"FromBytes", "load", "<16 x i8>", "1"},
		{"FromBytePointer", "load", "<16 x i8>", "1"},
		{"LoadBytes", "load", "<16 x i8>", "1"},
		{"StoreBytes", "store", "<16 x i8>", "1"},
		{"FromFloats", "load", "<4 x float>", "4"},
		{"LoadFloats", "load", "<4 x float>", "4"},
		{"StoreFloats", "store", "<4 x float>", "4"},
	} {
		fn := mod.NamedFunction("main." + tc.name)
		if fn.IsNil() {
			t.Fatal("missing", tc.name)
		}
		pattern := regexp.MustCompile(tc.op + ` ` + regexp.QuoteMeta(tc.vector) + `[^\n]*, align ` + tc.alignment + `(?:,|\n)`)
		if !pattern.MatchString(fn.String()) {
			t.Fatalf("%s: source alignment lost:\n%s", tc.name, fn.String())
		}
	}
	owned := mod.NamedGlobal("main.Owned")
	if owned.IsNil() || !strings.Contains(owned.String(), "align 16") {
		t.Fatalf("owned vector lost native alignment: %s", owned.String())
	}
	if target := pkg.Prog.Target(); target.GOOS != "linux" || target.GOARCH != "amd64" {
		return
	}
	// Unknown pointers must remain safe after optimization. Other loads from
	// proven aligned storage may legitimately regain stronger alignment.
	for _, level := range []string{"O0", "O2"} {
		if level == "O2" {
			opts := llvm.NewPassBuilderOptions()
			opts.SetVerifyEach(true)
			err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), opts)
			opts.Dispose()
			if err != nil {
				t.Fatal(err)
			}
		}
		asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
		if err != nil {
			t.Fatal(err)
		}
		text := string(asm.Bytes())
		asm.Dispose()
		for _, name := range []string{"LoadBytes", "StoreBytes", "LoadFloats", "StoreFloats"} {
			start := strings.Index(text, "main."+name+":")
			if start < 0 {
				t.Fatal("missing assembly", name)
			}
			body, _, _ := strings.Cut(text[start:], ".Lfunc_end")
			if !regexp.MustCompile(`mov(?:ups|dqu)\s`).MatchString(body) ||
				regexp.MustCompile(`mov(?:aps|dqa)\s+[^\n]*\(`).MatchString(body) {
				t.Fatalf("%s %s: expected unaligned vector memory operations:\n%s", level, name, body)
			}
		}
	}
}
