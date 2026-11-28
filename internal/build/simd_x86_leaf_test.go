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
func wide256(x archsimd.Float32x8) archsimd.Float32x8{return x.RoundScaled(2)}
func wide512(x archsimd.Float32x16) archsimd.Float32x16{return x.RoundScaled(2)}
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
				cases, lanes       int
			}{
				{"fixed", "avx512.mask.rndscale.ps.128_0_imm2", "i32 32,", 1, 4},
				{"dynamic", "avx512.mask.rndscale.ps.128_0", "", 16, 4},
				{"sha", "sha1rnds4_0_imm3", "i8 3)", 1, 4},
				{"shaDynamic", "sha1rnds4_0", "", 4, 4},
				{"wide256", "avx512.mask.rndscale.ps.256_0_imm2", "i32 32,", 1, 8},
				{"wide512", "avx512.mask.rndscale.ps.512_0_imm2", "i32 32,", 1, 16},
			} {
				name := "__llgo_simd_" + tc.sym
				leaf := m.NamedFunction(name)
				if leaf.IsNil() {
					t.Fatal("missing " + name)
				}
				assertDirectX86Leaf(t, m.NamedFunction("main."+tc.fn), name)
				ir := leaf.String()
				if count := strings.Count(ir, "@llvm.x86."); count != tc.cases {
					t.Fatalf("%s contains %d instruction calls, want %d", name, count, tc.cases)
				}
				if tc.immediate != "" && !strings.Contains(ir, tc.immediate) || strings.Contains(ir, "switch ") != (tc.cases > 1) {
					t.Fatalf("wrong immediate encoding in %s", ir)
				}
				ft := leaf.GlobalValueType()
				if ft.ReturnType().TypeKind() != llvm.VectorTypeKind || ft.ReturnType().VectorSize() != tc.lanes {
					t.Fatalf("wrong vector result in %s", ir)
				}
				for _, param := range ft.ParamTypes() {
					if param != ft.ReturnType() && !(tc.cases > 1 && param.TypeKind() == llvm.IntegerTypeKind) {
						t.Fatalf("unexpected vector carrier in %s", ir)
					}
				}
				if (osname == "windows") != strings.Contains(ir, "x86_vectorcallcc") {
					t.Fatalf("wrong vector calling convention on %s: %s", osname, ir)
				}
				for use := leaf.FirstUse(); !use.IsNil(); use = use.NextUse() {
					call := use.User()
					if call.IsACallInst().IsNil() || call.InstructionCallConv() != leaf.FunctionCallConv() {
						t.Fatalf("call does not match the vector leaf ABI: %s", call.String())
					}
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

func assertDirectX86Leaf(t *testing.T, fn llvm.Value, leaf string) {
	t.Helper()
	if fn.IsNil() {
		t.Fatal("missing caller of " + leaf)
	}
	body := fn.String()
	if !strings.Contains(body, "@"+leaf+"(") {
		t.Fatalf("%s has no call to %s:\n%s", fn.Name(), leaf, body)
	}
	for _, guard := range []string{"runtime.CPUFeatures", "SIMDRequireX86Features", "PanicSIMDX86Feature"} {
		if strings.Contains(body, guard) {
			t.Fatalf("%s has an implicit CPU check before %s:\n%s", fn.Name(), leaf, body)
		}
	}
}
