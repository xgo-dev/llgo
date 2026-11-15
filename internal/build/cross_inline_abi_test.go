package build

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestCrossPackageInlineAggregateABI(t *testing.T) {
	const dependency = "github.com/xgo-dev/llgo/benchmark/crossinline/value"
	methods := []string{"Mixed.Shift", "Mixed.Summary", "ScalarPair.Shift"}
	for _, target := range []struct{ os, arch, smallABI string }{
		{"linux", "amd64", "{ i64, double } (i64, double, i64)"},
		{"linux", "arm64", "[2 x i64] ([2 x i64], i64)"},
		{"darwin", "arm64", "[2 x i64] ([2 x i64], i64)"},
		{"windows", "386", "void (ptr, i64, double, i64)"},
		{"wasip1", "wasm", "void (ptr, ptr, i64)"},
	} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.OptLevel = target.os, target.arch, optlevel.Os
			conf.ForceRebuild = true
			var owner llssa.Package
			conf.ModuleHook = func(pkg Package) {
				if pkg.PkgPath == dependency {
					owner = pkg.LPkg
				}
			}
			pkgs, err := Do([]string{"./testdata/crossinlineabi"}, conf)
			if err != nil {
				t.Fatal(err)
			}
			caller := pkgs[0].LPkg
			defer caller.Prog.Dispose()
			if owner == nil {
				t.Fatal("dependency module missing")
			}
			mod := caller.Module()
			for _, method := range methods {
				name := dependency + "." + method
				imported, defined := mod.NamedFunction(name), owner.Module().NamedFunction(name)
				if imported.IsNil() || defined.IsNil() {
					t.Fatalf("missing method %s", name)
				}
				if imported.GlobalValueType() != defined.GlobalValueType() {
					t.Fatalf("%s: caller type %s differs from owner %s", name, imported.GlobalValueType(), defined.GlobalValueType())
				}
				// Hosted Wasm uses shadow stacks, which disable body import.
				if imported.IsDeclaration() != (target.arch == "wasm") {
					t.Fatalf("unexpected import policy for %s", name)
				}
				want := "void (ptr, ptr, i64)"
				if method == "Mixed.Summary" {
					want = "void (ptr, ptr)"
				} else if method == "ScalarPair.Shift" {
					want = target.smallABI
				}
				if got := imported.GlobalValueType().String(); got != want {
					t.Fatalf("%s: ABI = %s, want %s", name, got, want)
				}
				if imported.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind &&
					imported.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret")).IsNil() {
					t.Fatalf("%s: aggregate return lost sret", name)
				}
				for i := 1; i <= defined.ParamsCount(); i++ {
					for _, attr := range []string{"sret", "byval"} {
						kind := llvm.AttributeKindID(attr)
						a, b := imported.GetEnumAttributeAtIndex(i, kind), defined.GetEnumAttributeAtIndex(i, kind)
						if a.IsNil() != b.IsNil() || !a.IsNil() && a.GetTypeValue() != b.GetTypeValue() {
							t.Fatalf("%s: mismatched %s attribute on parameter %d", name, attr, i)
						}
					}
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			buf, err := caller.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			buf.Dispose()
		})
	}
}
