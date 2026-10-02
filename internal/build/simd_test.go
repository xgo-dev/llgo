//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

const simd128Source = `package main
import "simd/archsimd"
import "runtime"

//go:noinline
func add(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Add(y).Sub(y) }
//go:noinline
func bits(x, y archsimd.Uint64x2, i uint8) uint64 {
 z := x.Add(y).Sub(y).And(y).Or(x).Xor(y)
 return z.SetElem(i, x.GetElem(i)).GetElem(i)
}
type storedVector archsimd.Float32x4
func convert(x storedVector) archsimd.Float32x4 { return archsimd.Float32x4(x) }
//go:noinline
func identity(x archsimd.Float32x4) archsimd.Float32x4 { return x }
//go:noinline
func loop(x archsimd.Float32x4, n int) (archsimd.Float32x4, int) {
 for i := 0; i < n; i++ { x = x.Add(x) }
 return x, n
}
func load(p *[4]float32) archsimd.Float32x4 { return archsimd.LoadFloat32x4Array(p) }
func store(p *[4]float32, x archsimd.Float32x4) { x.StoreArray(p) }
func arithmetic(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Mul(y).Div(y).Sqrt().Round() }
func bitcast(x archsimd.Uint32x4) archsimd.Float32x4 { return x.BitsToFloat32() }
func abs(x archsimd.Int32x4) archsimd.Int32x4 { return x.Abs() }
func minmax(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Min(y).Max(y) }
func round32(x archsimd.Float32x4) archsimd.Float32x4 { return x.Round() }
func round64(x archsimd.Float64x2) archsimd.Float64x2 { return x.Round() }
func compare(x, y archsimd.Float32x4) archsimd.Mask32x4 { return x.Equal(y) }
func maskpass(x archsimd.Mask32x4) (archsimd.Mask32x4, int) { return x, 1 }
func maskbits(x archsimd.Mask32x4) archsimd.Int32x4 { return x.ToInt32x4() }
func shift(x archsimd.Int32x4, n uint64) archsimd.Int32x4 { return x.ShiftAllLeft(n).ShiftAllRight(n) }
func saturated(x, y archsimd.Int8x16) archsimd.Int8x16 { return x.AddSaturated(y).SubSaturated(y).Min(y).Max(y) }
func fixed(x archsimd.Float32x4) float32 { return x.GetElem(1) }
func boxed(x any) archsimd.Float32x4 { return x.(archsimd.Float32x4) }
func invoke(x, y archsimd.Float32x4) { defer x.Add(y); go x.Sub(y) }
func main() {
 _ = runtime.FuncForPC(0)
 var x archsimd.Float32x4
 _ = add(x, x)
 x, _ = loop(identity(convert(storedVector(x))), 2)
 invoke(x, x)
 var y archsimd.Uint64x2
 _ = bits(y, y, 0)
}
`

const simdMaskBitmapSource = `package main
import "simd/archsimd"
func maskFrom8(x uint16) archsimd.Mask8x16 { return archsimd.Mask8x16FromBits(x) }
func maskTo8(x archsimd.Mask8x16) uint16 { return x.ToBits() }
func maskFrom16(x uint8) archsimd.Mask16x8 { return archsimd.Mask16x8FromBits(x) }
func maskTo16(x archsimd.Mask16x8) uint8 { return x.ToBits() }
func maskFrom32(x uint8) archsimd.Mask32x4 { return archsimd.Mask32x4FromBits(x) }
func maskTo32(x archsimd.Mask32x4) uint8 { return x.ToBits() }
func maskFrom64(x uint8) archsimd.Mask64x2 { return archsimd.Mask64x2FromBits(x) }
func maskTo64(x archsimd.Mask64x2) uint8 { return x.ToBits() }
`

func simdTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"go.mod": "module simdtest\n\ngo 1.27\n", "main.go": simd128Source, "bitmap_amd64.go": simdMaskBitmapSource} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSIMD128LLVM(t *testing.T) {
	dir := simdTestDir(t)
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"linux", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 {
				t.Fatalf("packages: %d", len(pkgs))
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			fn := mod.NamedFunction("main.add")
			if fn.IsNil() || !strings.Contains(fn.String(), "fadd <4 x float>") {
				t.Fatal("Float32x4.Add did not lower to vector fadd")
			}
			if strings.Contains(fn.String(), "extractvalue") || strings.Contains(fn.String(), "insertelement") || strings.Contains(fn.String(), "extractelement") {
				t.Fatalf("arithmetic repacks vectors:\n%s", fn.String())
			}
			if fixed := mod.NamedFunction("main.fixed").String(); strings.Contains(fixed, "PanicSIMDImmediate") || strings.Contains(fixed, "br ") {
				t.Fatalf("valid constant lane retained a bounds branch:\n%s", fixed)
			}
			for _, name := range []string{"load", "store"} {
				ir := mod.NamedFunction("main." + name).String()
				if !strings.Contains(ir, name+" <4 x float>") || !strings.Contains(ir, "align 4") {
					t.Fatalf("%s does not use element-aligned vector memory:\n%s", name, ir)
				}
			}
			for name, instructions := range map[string][]string{
				"arithmetic": {"fmul <4 x float>", "fdiv <4 x float>", "@llvm.sqrt.v4f32"},
				"bitcast":    {"bitcast <4 x i32>", "to <4 x float>"},
				"abs":        {"@llvm.abs.v4i32", "i1 false"},
				"shift":      {"icmp uge i64", "shl <4 x i32>", "ashr <4 x i32>"},
				"saturated":  {"@llvm.sadd.sat.v16i8", "@llvm.ssub.sat.v16i8", "@llvm.smin.v16i8", "@llvm.smax.v16i8"},
			} {
				ir := mod.NamedFunction("main." + name).String()
				for _, instruction := range instructions {
					if !strings.Contains(ir, instruction) {
						t.Fatalf("%s missing %s:\n%s", name, instruction, ir)
					}
				}
			}
			if target.arch != "amd64" {
				ir := mod.NamedFunction("main.minmax").String()
				if !strings.Contains(ir, "@llvm.minimum.v4f32") || !strings.Contains(ir, "@llvm.maximum.v4f32") {
					t.Fatalf("missing IEEE vector min/max:\n%s", ir)
				}
			}
			if target.arch != "amd64" && !strings.Contains(mod.NamedFunction("main.round32").String(), "@llvm.roundeven.v4f32") {
				t.Fatal("missing native vector roundeven")
			}
			if ir := mod.NamedFunction("main.compare").String(); !strings.Contains(ir, "fcmp oeq <4 x float>") || !strings.Contains(ir, "sext <4 x i1>") {
				t.Fatalf("comparison does not produce canonical vector mask:\n%s", ir)
			}
			maskpass := mod.NamedFunction("main.maskpass")
			params := maskpass.GlobalValueType().ParamTypes()
			if params[len(params)-1].TypeKind() != llvm.VectorTypeKind || !strings.Contains(maskpass.String(), "sret({ <4 x i32>, i64 })") {
				t.Fatalf("mask loses vector ABI across multiple-result calls:\n%s", maskpass.String())
			}
			if target.arch == "amd64" {
				for _, name := range []string{"maskFrom8", "maskFrom16", "maskFrom32", "maskFrom64", "maskTo8", "maskTo16", "maskTo32", "maskTo64"} {
					ir := mod.NamedFunction("main." + name).String()
					if strings.Contains(ir, "call ") || !strings.Contains(ir, "bitcast") {
						t.Fatalf("mask bitmap conversion uses an external call:\n%s", ir)
					}
				}
			}
			identity := mod.NamedFunction("main.identity")
			if identity.GlobalValueType().ReturnType().TypeKind() != llvm.VectorTypeKind || identity.GlobalValueType().ParamTypes()[0].TypeKind() != llvm.VectorTypeKind {
				t.Fatalf("identity does not use a vector ABI:\n%s", identity.String())
			}
			if !strings.Contains(mod.NamedFunction("main.loop").String(), "phi <4 x float>") {
				t.Fatal("missing vector phi")
			}
			if target.arch == "wasm" {
				// Attributes are printed separately by LLVM; inspect the function itself.
				found := false
				for _, attr := range identity.GetFunctionAttributes() {
					if attr.IsString() && attr.GetStringKind() == "target-features" && strings.Contains(attr.GetStringValue(), "+simd128") {
						found = true
					}
				}
				if !found {
					t.Fatal("identity lacks wasm SIMD feature")
				}
			}
			prog := pkgs[0].LPkg.Prog
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(prog.Target().Spec().Triple)
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			opts.SetVerifyEach(true)
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			asm, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer asm.Dispose()
			want := map[string]string{"amd64": "addps", "arm64": "fadd", "wasm": "f32x4.add"}[target.arch]
			if !strings.Contains(string(asm.Bytes()), want) {
				t.Fatalf("missing %s in assembly", want)
			}
			if target.arch == "amd64" && strings.Contains(string(asm.Bytes()), "roundeven") {
				t.Fatal("baseline rounding requires nonportable libm roundeven")
			}

		})
	}
}

func TestSIMDIntrinsicDefinitions(t *testing.T) {
	dir := simdTestDir(t)
	for _, target := range []struct{ os, arch string }{{"windows", "amd64"}, {"windows", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			pkgs, err := Build(Invocation{Args: []string{".", "simd/archsimd"}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, pkg := range pkgs {
				defer pkg.LPkg.Prog.Dispose()
				if pkg.PkgPath != "simd/archsimd" {
					continue
				}
				found = true
				mod := pkg.LPkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if fn := mod.NamedFunction("simd/archsimd.Float32x4.ConvertToInt32"); fn.IsNil() || !strings.Contains(fn.String(), "PanicSIMDUnimplemented") {
					t.Fatal("missing explicit unsupported implementation")
				}
				if fn := mod.NamedFunction("simd/archsimd.Float32x4.Add"); fn.IsNil() || !strings.Contains(fn.String(), "fadd <4 x float>") {
					t.Fatal("implemented intrinsic has no callable definition")
				}
				for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
					if strings.HasPrefix(fn.Name(), "simd/archsimd.") && fn.IsDeclaration() && !fn.FirstUse().IsNil() {
						t.Errorf("intrinsic lacks a definition: %s", fn.Name())
					}
				}
			}
			if !found {
				t.Fatal("archsimd package was not compiled")
			}
		})
	}
}

func TestSIMDWindowsLinkname(t *testing.T) {
	dir := simdTestDir(t)
	const source = `package main
import "simd/archsimd"
import _ "unsafe"
//go:linkname broadcast simd/archsimd.BroadcastFloat32x4
func broadcast(float32) archsimd.Float32x4
func main() { println(broadcast(1).GetElem(0)) }
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOEXPERIMENT = "windows", "amd64", "simd"
	pkgs, err := Build(Invocation{Args: []string{".", "simd/archsimd"}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	for _, pkg := range pkgs {
		if pkg.PkgPath != "simd/archsimd" {
			continue
		}
		mod := pkg.LPkg.Module()
		fn := mod.NamedFunction("simd/archsimd.BroadcastFloat32x4")
		if fn.IsNil() || !strings.Contains(fn.String(), "broadcast1To4") {
			t.Fatal("linkname target body was discarded")
		}
		callee := mod.NamedFunction("simd/archsimd.Float32x4.broadcast1To4")
		if callee.IsNil() || callee.IsDeclaration() || !strings.Contains(callee.String(), "shufflevector") {
			t.Fatal("transitive broadcast intrinsic lacks a vector implementation")
		}
		return
	}
	t.Fatal("missing archsimd package")
}
