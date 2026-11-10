// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package cl

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

const sourceLocationProgram = `package location
import _ "unsafe"
//go:linkname FILE llgo.file
var FILE *int8
//go:linkname LINE llgo.line
var LINE int32
//line /generated/input.xgo:100
var Initial = LINE
func Line() int32 { return LINE }
func Pair() (int32,int32) { return LINE, LINE }
func File() *int8 { return FILE }
func OtherFile() *int8 { return FILE }
func Closure() func() int32 { return func() int32 { return LINE } }
func Generic[T any](v T) int32 { return LINE }
func Instance() int32 { return Generic(1) }
func Shadow() int32 { LINE := int32(7); LINE++; return LINE }
`

func TestSourceLocationLLVM(t *testing.T) {
	for _, target := range []llssa.Target{{GOOS: "linux", GOARCH: "386"}, {GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "wasip1", GOARCH: "wasm"}} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, sourceLocationProgram)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			defer pkg.Module().Dispose()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			ir := pkg.String()
			for _, bad := range []string{"@llgo.file", "@llgo.line", "@location.FILE", "@location.LINE"} {
				if strings.Contains(ir, bad) {
					t.Errorf("emitted compiler variable storage %s\n%s", bad, ir)
				}
			}
			if strings.Count(ir, `c"/generated/input.xgo\00"`) != 1 {
				t.Fatalf("file constant not deduplicated/NUL terminated:\n%s", ir)
			}
			for _, test := range []struct{ name, want string }{{"Line", "ret i32 101"}, {"Shadow", "ret i32 8"}} {
				body := mustNamedFunction(t, pkg.Module(), "location."+test.name).String()
				if !strings.Contains(body, test.want) {
					t.Errorf("%s missing %q:\n%s", test.name, test.want, body)
				}
			}
			// The two reads on one source line are independent constants of C int width.
			pair := mustNamedFunction(t, pkg.Module(), "location.Pair").String()
			if strings.Contains(pair, "load i32") || !strings.Contains(pair, "i32 102") {
				t.Errorf("pair:\n%s", pair)
			}
			if !strings.Contains(ir, "i32 106") {
				t.Errorf("generic body location lost:\n%s", ir)
			}
		})
	}
}

func TestSourceLocationRejectErasedAddress(t *testing.T) {
	for _, body := range []string{"func f() int32 { return *(&LINE) }", "func f() { if false { LINE++ } }"} {
		ssapkg, _, files := buildGoSSAPkg(t, "package p\nimport _ \"unsafe\"\n//go:linkname LINE llgo.line\nvar LINE int32\n"+body)
		prog := newLLSSAProg(t)
		_, err := NewPackage(prog, ssapkg, files)
		prog.Dispose()
		if err == nil || !strings.Contains(err.Error(), "compiler variable LINE") {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestSourceLocationEffectivePatch(t *testing.T) {
	for _, replacementSpecial := range []bool{false, true} {
		t.Run(fmt.Sprint(replacementSpecial), func(t *testing.T) {
			source := func(special bool) string {
				decl := "package location\nimport _ \"unsafe\"\n"
				if special {
					decl += "//go:linkname Line llgo.line\n"
				}
				return decl + "var Line int32\n"
			}
			original, ofset, ofiles := buildGoSSAPkg(t, source(!replacementSpecial))
			alternate, afset, afiles := buildGoSSAPkg(t, source(replacementSpecial))
			prog := newLLSSAProg(t)
			defer prog.Dispose()
			if err := ParsePkgSyntax(prog, ofset, original.Pkg, ofiles); err != nil {
				t.Fatal(err)
			}
			if err := ParsePkgSyntax(prog, afset, alternate.Pkg, afiles); err != nil {
				t.Fatal(err)
			}
			ctx := &context{prog: prog, patches: Patches{"location": {Types: alternate.Pkg}}}
			global := original.Var("Line")
			kind := ctx.compilerVariable(global).Kind
			if (kind == directive.LineVariable) != replacementSpecial {
				t.Fatalf("effective kind=%v", kind)
			}
			_, vtype, define := ctx.varName(original.Pkg, global)
			if replacementSpecial {
				if vtype != ignoredVar || define {
					t.Fatal("replacement gained storage")
				}
			} else if vtype != goVar || !define {
				t.Fatal("replacement inherited original link")
			}
		})
	}
}
