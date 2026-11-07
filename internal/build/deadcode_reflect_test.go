package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestReflectNameClones(t *testing.T) {
	var a reflectNameAccumulator
	a.add("resolved", []string{"BeforeSave"}, true)
	a.add("resolved", []string{"AfterSave", "BeforeSave"}, true)
	a.add("partial", []string{"BeforeSave"}, true)
	a.add("partial", nil, false)
	a.add("unknown-first", nil, false)
	a.add("unknown-first", []string{"AfterSave"}, true)
	a.add("empty", nil, true)
	want := map[string][]string{"resolved": {"AfterSave", "BeforeSave"}}
	if got := a.result(); !reflect.DeepEqual(got, want) {
		t.Fatalf("cloned callsite names = %v, want %v", got, want)
	}
}

func TestReflectPrivateGlobalVisibility(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	for _, mode := range []string{"private", "backing", "exported", "missing source", "annotation", "alias", "target", "locality", "foreign definition", "foreign reference", "missing module"} {
		t.Run(mode, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			pkg := types.NewPackage("example.com/model", "model")
			for _, name := range []string{"callbacks", "Exported"} {
				pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, name, types.NewSlice(types.Typ[types.String])))
			}
			owner := &aPackage{Package: &packages.Package{Types: pkg}, LPkg: prog.NewPackage(pkg.Path(), pkg.Name())}
			other := &aPackage{LPkg: prog.NewPackage("other", "other")}
			ctx := &context{prog: prog}
			name := "example.com/model.callbacks"
			exported := make(map[string]bool)
			switch mode {
			case "backing":
				name += "$data"
			case "exported":
				name = "example.com/model.Exported"
			case "missing source":
				name += "$unknown"
			case "annotation":
				exported[name] = true
			case "alias":
				prog.SetLinkname(name, "external")
			case "target":
				prog.SetLinkname("other.alias", name)
			case "locality":
				prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
			case "foreign definition", "foreign reference":
				mod := other.LPkg.Module()
				g := llvm.AddGlobal(mod, mod.Context().Int64Type(), name)
				if mode == "foreign definition" {
					g.SetInitializer(llvm.ConstInt(g.GlobalValueType(), 0, false))
				} else {
					ref := llvm.AddGlobal(mod, g.Type(), "other.ref")
					ref.SetInitializer(g)
				}
			case "missing module":
				other.LPkg = nil
			}
			want := mode == "private" || mode == "backing"
			if got := privateReflectGlobal(ctx, []Package{owner, other}, owner, name, exported); got != want {
				t.Fatalf("private global proof = %v, want %v", got, want)
			}
		})
	}
}

func TestReflectNativeVisibility(t *testing.T) {
	for _, tc := range []struct {
		name, flags, source, native string
		want                        bool
	}{
		{"plain", "", "", "", true},
		{"backend selection", "-Wl,-no_exported_symbols --ld-path=/opt/lld/ld64.lld -fuse-ld=lld", "", "", true},
		{"mixed flag", "-Wl,-no_exported_symbols,-force_load,other.a", "", "", false},
		{"extra input", "other.a", "", "", false},
		{"system import", "", "//go:cgo_import_dynamic libc_write write \"/usr/lib/libSystem.B.dylib\"", "", true},
		{"custom import", "", "//go:cgo_import_dynamic write write \"/custom/lib.dylib\"", "", false},
		{"custom cgo flag", "", "//go:cgo_ldflag \"-lcustom\"", "", false},
		{"native object", "", "", "extra.o", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LDFLAGS", tc.flags)
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", "package p\n"+tc.source+"\n", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &packages.Package{PkgPath: "p", Dir: t.TempDir(), Syntax: []*ast.File{file}}
			if tc.native != "" {
				pkg.OtherFiles = []string{tc.native}
			}
			ctx := &context{buildConf: &Config{Goos: "darwin", Goarch: "arm64"}, pkgByID: map[string]*aPackage{"p": {Package: pkg}}}
			got := opaqueGlobalInitPackageWithPolicy(ctx, make(map[string]bool), reflectGlobalVisibilityPolicy(ctx)) == ""
			if got != tc.want {
				t.Fatalf("native visibility = %v, want %v", got, tc.want)
			}
			if tc.name == "system import" && opaqueGlobalInitPackage(ctx, make(map[string]bool)) == "" {
				t.Fatal("reflection policy changed the existing initializer-pruning policy")
			}
		})
	}
}
