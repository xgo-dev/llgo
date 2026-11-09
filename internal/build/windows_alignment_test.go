package build

import (
	"go/types"
	"path/filepath"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWindowsFaultContextAlignment(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, tc := range []struct {
		arch string
		size int64
	}{{"amd64", 1232}, {"arm64", 912}} {
		t.Run(tc.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch = "windows", tc.arch
			pkgs, err := Build(Invocation{Args: []string{"runtime"}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			pkg := pkgs[0]
			defer pkg.LPkg.Prog.Dispose()
			if pkg.AltPkg == nil {
				t.Fatal("missing replacement runtime")
			}
			obj := pkg.AltPkg.Types.Scope().Lookup("windowsFaultContextStorage")
			if obj == nil {
				t.Fatal("missing CONTEXT storage")
			}
			typ := obj.Type()
			prog := pkg.LPkg.Prog
			sizes := prog.TypeSizes(types.SizesFor("gc", tc.arch))
			fields := typ.Underlying().(*types.Struct)
			offsets := sizes.Offsetsof([]*types.Var{fields.Field(0), fields.Field(1)})
			if sizes.Alignof(typ) != 16 || sizes.Sizeof(typ) != tc.size || offsets[1] != 0 {
				t.Fatalf("CONTEXT storage: align=%d size=%d offset=%d", sizes.Alignof(typ), sizes.Sizeof(typ), offsets[1])
			}
			for _, name := range []string{"fpCallers", "platformFaultCallers", "recoverFrameMarks", "recoverFrameLive"} {
				fn := pkg.LPkg.Module().NamedFunction("runtime." + name)
				if fn.IsNil() {
					t.Fatalf("missing function %s", name)
				}
				found := false
				for _, bb := range fn.BasicBlocks() {
					for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
						if !inst.IsACallInst().IsNil() && inst.CalledValue().Name() == llssa.PkgRuntime+".AllocZAligned" {
							if inst.Operand(0).ZExtValue() == uint64(tc.size) && inst.Operand(1).ZExtValue() == 16 {
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatalf("%s: missing aligned CONTEXT allocation: %s", name, fn.String())
				}
			}
			storage := prog.Type(typ, llssa.GoDecl)
			if prog.AlignOf(storage) != 16 || prog.SizeOf(storage) != uint64(tc.size) {
				t.Fatal("LLVM CONTEXT storage differs from source layout")
			}
		})
	}
}
