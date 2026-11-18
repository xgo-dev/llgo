package cl

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestGoArrayAccessAlignment(t *testing.T) {
	const source = `package arrayalignment
 type A [2]int64
 type N [2]A
 type S struct { B byte; V A }
 type T struct { B byte; V N }
 func Read(p *S) A { return p.V }
 func Write(p *S, v A) { p.V = v }
 func Address(p *S) *A { return &p.V }
 func ReadPtr(p *A) A { return *p }
 func WritePtr(p *A, v A) { *p = v }
 func ReadNested(p *T) N { return p.V }
 func WriteNested(p *T, v N) { p.V = v }
 func AddressNested(p *T) *N { return &p.V }
 func ReadNestedPtr(p *N) N { return *p }
 func WriteNestedPtr(p *N, v N) { *p = v }
 `
	for _, target := range []llssa.Target{{GOOS: "windows", GOARCH: "386"}, {GOOS: "linux", GOARCH: "386"}, {GOOS: "linux", GOARCH: "amd64"}} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			defer pkg.Module().Dispose()
			want := prog.GoWordSize()
			for _, name := range []string{"S", "T"} {
				typ := prog.Type(ssaPkg.Pkg.Scope().Lookup(name).Type(), llssa.GoDecl)
				if offset, alignment := prog.OffsetOf(typ, 1), prog.AlignOf(typ); offset != uint64(want) || alignment != uint64(want) {
					t.Errorf("%s field-offset/struct-alignment = %d/%d, want %d/%d", name, offset, alignment, want, want)
				}
			}
			// Address/AddressNested let field pointers escape to callers. The pointer
			// consumers must honor the same alignment even without a visible FieldAddr.
			for _, name := range []string{"Read", "Write", "ReadPtr", "WritePtr", "ReadNested", "WriteNested", "ReadNestedPtr", "WriteNestedPtr"} {
				t.Run(name, func(t *testing.T) {
					fn := mustNamedFunction(t, pkg.Module(), "arrayalignment."+name)
					accesses := 0
					for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
						for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
							var typ llvm.Type
							if !inst.IsALoadInst().IsNil() {
								typ = inst.Type()
							} else if !inst.IsAStoreInst().IsNil() {
								typ = inst.Operand(0).Type()
							} else {
								continue
							}
							if typ.TypeKind() != llvm.ArrayTypeKind {
								continue
							}
							accesses++
							if inst.Alignment() != want {
								t.Errorf("array access must use alignment %d: %s", want, inst.String())
							}
						}
					}
					if accesses != 1 {
						t.Errorf("got %d array accesses, want 1:\n%s", accesses, fn.String())
					}
				})
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			obj, err := prog.TargetMachine().EmitToMemoryBuffer(pkg.Module(), llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			obj.Dispose()
		})
	}
}
