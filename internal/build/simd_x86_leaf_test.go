//go:build !llgo

package build

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDX86SpecialLeafABI(t *testing.T) {
	const src = `package main
import "simd/archsimd"
func fixed(o,a *[4]float32){archsimd.LoadFloat32x4Array(a).RoundScaled(18).StoreArray(o)}
func dynamic(o,a *[4]float32,k uint8){archsimd.LoadFloat32x4Array(a).RoundScaled(k).StoreArray(o)}
func sha(o,a *[4]uint32){x:=archsimd.LoadUint32x4Array(a);x.SHA1FourRounds(7,x).StoreArray(o)}
func shaDynamic(o,a *[4]uint32,k uint8){x:=archsimd.LoadUint32x4Array(a);x.SHA1FourRounds(k,x).StoreArray(o)}
func wide(o,a *[16]float32){archsimd.LoadFloat32x16Array(a).RoundScaled(2).StoreArray(o)}
func main(){}
`
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module leaf\n\ngo 1.27\n")
	writeFile(t, filepath.Join(dir, "main.go"), src)
	for _, osname := range []string{"linux", "windows"} {
		t.Run(osname, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = osname, "amd64", "v1", "simd"
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			m := pkgs[0].LPkg.Module()
			if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				fn, sym, immediate string
				mask               uint64
				cases              int
				wide               bool
			}{
				{"fixed", "avx512.mask.rndscale.ps.128_0_imm2", "i32 32,", 256, 1, false},
				{"dynamic", "avx512.mask.rndscale.ps.128_0", "", 256, 16, false},
				{"sha", "sha1rnds4_0_imm3", "i8 3)", 128, 1, false},
				{"shaDynamic", "sha1rnds4_0", "", 128, 4, false},
				{"wide.__llgo_fmv_guard_base", "avx512.mask.rndscale.ps.512_0_imm2", "i32 32,", 256, 1, true},
			} {
				name := "__llgo_simd_" + tc.sym
				leaf := m.NamedFunction(name)
				if leaf.IsNil() {
					t.Fatal("missing " + name)
				}
				assertX86LeafGuard(t, m.NamedFunction("main."+tc.fn), name, tc.mask)
				ir := leaf.String()
				if count := strings.Count(ir, "@llvm.x86."); count != tc.cases {
					t.Fatalf("%s contains %d instruction calls, want %d", name, count, tc.cases)
				}
				if tc.immediate != "" && !strings.Contains(ir, tc.immediate) || strings.Contains(ir, "switch ") != (tc.cases > 1) {
					t.Fatalf("wrong immediate encoding in %s", ir)
				}
				ft := leaf.GlobalValueType()
				if tc.wide || osname == "windows" {
					if ft.ReturnType().TypeKind() != llvm.VoidTypeKind {
						t.Fatalf("expected a caller-owned result in %s", ir)
					}
					for _, p := range ft.ParamTypes() {
						if p.TypeKind() != llvm.PointerTypeKind && !(tc.cases > 1 && p.TypeKind() == llvm.IntegerTypeKind) {
							t.Fatal(ir)
						}
					}
				} else if ft.ReturnType().TypeKind() != llvm.VectorTypeKind || strings.Contains(ir, "ptr ") {
					t.Fatal(ir)
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, pkgs[:1], func(t *testing.T, mod llvm.Module, asm string) {
				for _, op := range []string{"vrndscaleps", "sha1rnds4"} {
					if !strings.Contains(asm, op) {
						t.Error("missing", op)
					}
				}
			})
		})
	}
}

func assertX86LeafGuard(t *testing.T, fn llvm.Value, leafName string, mask uint64) {
	t.Helper()
	if fn.IsNil() {
		t.Fatal("missing x86 leaf caller")
	}
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		br := bb.LastInstruction()
		if br.InstructionOpcode() != llvm.Br || br.SuccessorsCount() != 2 {
			continue
		}
		cmp := br.Operand(0).IsAICmpInst()
		if cmp.IsNil() || cmp.IntPredicate() != llvm.IntNE {
			continue
		}
		and := cmp.Operand(0).IsABinaryOperator()
		if and.IsNil() || and.InstructionOpcode() != llvm.And {
			continue
		}
		bit := llvm.ConstInt(and.Type(), mask, false)
		load := and.Operand(0).IsALoadInst()
		if load.IsNil() || load.Operand(0).Name() != "github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures" || and.Operand(1) != bit || cmp.Operand(1) != bit {
			continue
		}
		if strings.Contains(br.Successor(0).AsValue().String(), "PanicSIMDX86Feature") && strings.Contains(br.Successor(1).AsValue().String(), "@"+leafName+"(") {
			return
		}
	}
	t.Fatalf("%s has no checked call to %s for mask %#x:\n%s", fn.Name(), leafName, mask, fn.String())
}
