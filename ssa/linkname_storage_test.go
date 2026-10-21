package ssa

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLinknameTargetRebinding(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.SetLinkname("p.A", "shared")
	prog.SetLinkname("p.B", "shared")
	prog.SetLinkname("p.A", "shared") // Idempotent updates must not leak counts.
	prog.SetLinkname("p.A", "other")
	if !prog.HasLinknameTarget("shared") || !prog.HasLinknameTarget("other") {
		t.Fatal("rebinding one alias removed a live target")
	}
	prog.SetLinkname("p.B", "other")
	if prog.HasLinknameTarget("shared") || !prog.HasLinknameTarget("other") {
		t.Fatal("last alias did not remove the old target")
	}
	if target, ok := prog.Linkname("p.A"); !ok || target != "other" {
		t.Fatalf("forward index = %q, %v", target, ok)
	}
}

func TestSharedGoStorageDefinitions(t *testing.T) {
	for _, target := range []*Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "wasip1", GOARCH: "wasm"}} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			pkg := prog.NewPackage("storage", "storage")
			ptr := types.NewPointer(types.Typ[types.Int32])
			strong := prog.IntVal(41, prog.Type(types.Typ[types.Int32], GoDecl))
			for _, order := range []string{"zero-first", "static-first"} {
				t.Run(order, func(t *testing.T) {
					g := pkg.NewVar(order, ptr, GoDecl)
					values := []Expr{{}, strong}
					if order == "static-first" {
						values[0], values[1] = values[1], values[0]
					}
					for _, value := range values {
						if err := g.InitSharedGo(value, false); err != nil {
							t.Fatal(err)
						}
					}
					if g.impl.Initializer().ZExtValue() != 41 || g.impl.Linkage() != llvm.ExternalLinkage {
						t.Fatal("zero storage overrode the static definition")
					}
					if err := g.InitSharedGo(strong, false); err == nil || !strings.Contains(err.Error(), order) {
						t.Fatalf("duplicate static definition diagnostic = %v", err)
					}
				})
			}
			zero := pkg.NewVar("zero", ptr, GoDecl)
			for range 2 {
				if err := zero.InitSharedGo(Expr{}, false); err != nil {
					t.Fatal(err)
				}
			}
			want := llvm.CommonLinkage
			if target.GOARCH == "wasm" || target.GOOS == "darwin" {
				want = llvm.WeakAnyLinkage
			}
			if zero.impl.Initializer().IsNil() || !zero.impl.Initializer().IsNull() || zero.impl.Linkage() != want {
				t.Fatal("zero declarations must define coalescible storage")
			}
			imported := pkg.NewVar("imported", ptr, GoDecl)
			if err := imported.InitSharedGo(Expr{}, true); err != nil {
				t.Fatal(err)
			}
			if !imported.impl.Initializer().IsNil() {
				t.Fatal("cgo import acquired zero storage")
			}
			if err := imported.InitSharedGo(strong, true); err != nil {
				t.Fatal(err)
			}
			if imported.impl.Initializer().ZExtValue() != 41 {
				t.Fatal("cgo import replaced static data")
			}
			empty := pkg.NewVar("empty", types.NewPointer(types.NewStruct(nil, nil)), GoDecl)
			if err := empty.InitSharedGo(Expr{}, false); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
