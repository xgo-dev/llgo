//go:build !llgo

package build

import (
	"os"
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
func wide(o,a *[16]float32){archsimd.LoadFloat32x16Array(a).RoundScaled(2).StoreArray(o)}
func main(){}
`
	for _, osname := range []string{"linux", "windows"} {
		t.Run(osname, func(t *testing.T) {
			dir := t.TempDir()
			for name, data := range map[string]string{"go.mod": "module leaf\n\ngo 1.27\n", "main.go": src} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
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
				fn, sym       string
				dynamic, wide bool
			}{
				{"fixed", "avx512.mask.rndscale.ps.128_0_imm2", false, false},
				{"dynamic", "avx512.mask.rndscale.ps.128_0", true, false},
				{"sha", "sha1rnds4_0_imm3", false, false},
				{"wide", "avx512.mask.rndscale.ps.512_0_imm2", false, true},
			} {
				name := "__llgo_simd_" + tc.sym
				leaf := m.NamedFunction(name)
				if leaf.IsNil() {
					t.Fatal("missing " + name)
				}
				caller := m.NamedFunction("main." + tc.fn).String()
				guard := strings.Index(caller, ".CPUFeatures")
				call := strings.Index(caller, name+"(")
				if guard < 0 || call < guard || !strings.Contains(caller, "PanicSIMDX86Feature") || !strings.Contains(caller, "br i1") {
					t.Fatal("missing checked call", caller)
				}
				if strings.Contains(leaf.String(), "switch ") != tc.dynamic {
					t.Fatal(leaf.String())
				}
				ft := leaf.GlobalValueType()
				if tc.wide || osname == "windows" {
					for _, p := range ft.ParamTypes() {
						if p.TypeKind() != llvm.PointerTypeKind && !(tc.dynamic && p.TypeKind() == llvm.IntegerTypeKind) {
							t.Fatal(leaf.String())
						}
					}
				} else if ft.ReturnType().TypeKind() != llvm.VectorTypeKind || strings.Contains(leaf.String(), "ptr ") {
					t.Fatal(leaf.String())
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, pkgs[:1], func(t *testing.T, asm string) {
				for _, op := range []string{"vrndscaleps", "sha1rnds4"} {
					if !strings.Contains(asm, op) {
						t.Error("missing", op)
					}
				}
			})
		})
	}
}
