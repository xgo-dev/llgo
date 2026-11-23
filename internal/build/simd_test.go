//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"regexp"
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
func conversion(x archsimd.Float32x4) archsimd.Float32x4 { return x.ConvertToInt32().ConvertToFloat32() }
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
	for name, text := range map[string]string{
		"go.mod": "module simdtest\n\ngo 1.27\n", "main.go": simd128Source, "bitmap_amd64.go": simdMaskBitmapSource,
		"lookup_arm64.go": `package main; import "simd/archsimd"; func lookup(x,y archsimd.Int8x16) archsimd.Int8x16 { return x.LookupOrZero(y) }`,
		"lookup_wasm.go":  `package main; import "simd/archsimd"; func lookup(x,y archsimd.Int8x16) archsimd.Int8x16 { return x.LookupOrZero(y) }`,
		"permute_amd64.go": `package main; import "simd/archsimd"; func permute(x archsimd.Uint8x16, y archsimd.Int8x16) archsimd.Uint8x16 { return x.PermuteOrZero(y) }
func permuteConstant(x archsimd.Uint8x16) archsimd.Uint8x16 {
 indices := [16]uint8{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16}
 return x.Permute(archsimd.LoadUint8x16Array(&indices))
}
func permuteZeroConstant(x archsimd.Uint8x16) archsimd.Uint8x16 {
 indices := [16]int8{-1, 30, -128, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16}
 return x.PermuteOrZero(archsimd.LoadInt8x16Array(&indices))
}`,
	} {
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
			if target.arch == "amd64" {
				conf.GOAMD64 = "v1"
			}
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
			invoke := mod.NamedFunction("main.invoke")
			// Invocation-owned defer records use aligned stack storage on
			// native and explicit-root targets alike.
			found := false
			for bb := invoke.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
					if inst.IsAAllocaInst().IsNil() || !strings.Contains(inst.AllocatedType().String(), "archsimd.v128") {
						continue
					}
					found = true
					if inst.Alignment() < 16 {
						t.Fatalf("stack SIMD defer storage lacks 16-byte alignment:\n%s", inst.String())
					}
				}
			}
			if !found {
				t.Fatalf("missing stack SIMD defer storage:\n%s", invoke.String())
			}
			for _, name := range []string{"load", "store"} {
				ir := mod.NamedFunction("main." + name).String()
				if !strings.Contains(ir, name+" <4 x float>") || !strings.Contains(ir, "align 4") {
					t.Fatalf("%s does not use element-aligned vector memory:\n%s", name, ir)
				}
			}
			for name, instructions := range map[string][]string{
				"conversion": {"@llvm.fptosi.sat.v4i32.v4f32", "sitofp <4 x i32>"},
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
			if target.arch == "amd64" {
				for _, name := range []string{"permuteConstant", "permuteZeroConstant"} {
					ir := mod.NamedFunction("main." + name).String()
					if !strings.Contains(ir, "shufflevector") || strings.Contains(ir, "extractelement") || strings.Contains(ir, "insertelement") {
						t.Fatalf("constant permutation did not fold to a vector shuffle:\n%s", ir)
					}
				}
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
			if target.arch == "amd64" {
				assertSIMDBaselineAssembly(t, mod, string(asm.Bytes()))
			}
			if target.arch != "amd64" {
				wantLookup := map[string]string{"arm64": "tbl", "wasm": "i8x16.swizzle"}[target.arch]
				if !strings.Contains(string(asm.Bytes()), wantLookup) {
					t.Fatalf("missing lookup instruction %s", wantLookup)
				}
			}
			if target.arch == "amd64" && strings.Contains(string(asm.Bytes()), "roundeven") {
				t.Fatal("baseline rounding requires nonportable libm roundeven")
			}

		})
	}
}

// Check every Linux function compiled for the baseline, including scalar
// dispatchers. Wide signatures and guarded variants declare their own ISA.
func assertSIMDBaselineAssembly(t *testing.T, mod llvm.Module, asm string) {
	t.Helper()
	bodies := regexp.MustCompile(`(?ms)^\s*\.type\s+([^\n]+),@function\n(.*?)^\s*\.size\s+[^\n]+`).FindAllStringSubmatch(asm, -1)
	if len(bodies) == 0 {
		t.Fatal("no assembly functions found")
	}
	avx := regexp.MustCompile(`(?m)^\s+(?:v[a-z][a-z0-9]*|pclmul[a-z0-9]*)\s`)
	for _, body := range bodies {
		fn := mod.NamedFunction(strings.Trim(body[1], "\""))
		if !fn.IsNil() {
			attr := fn.GetStringAttributeAtIndex(-1, "target-features")
			if !attr.IsNil() && strings.Contains(attr.GetStringValue(), "+avx") {
				continue
			}
		}
		if avx.MatchString(body[2]) {
			t.Fatalf("GOAMD64=v1 baseline %s emitted optional vector instructions:\n%s", body[1], body[2])
		}
	}
}

func TestSIMDIntrinsicDefinitions(t *testing.T) {
	dir := simdTestDir(t)
	for _, target := range []struct{ os, arch string }{{"windows", "amd64"}, {"windows", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			if target.arch == "amd64" {
				conf.GOAMD64 = "v1"
			}
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
				if fn := mod.NamedFunction("simd/archsimd.Int8x16.OnesCount"); fn.IsNil() || !strings.Contains(fn.String(), "llvm.ctpop") {
					t.Fatal("missing callable population count")
				}
				if target.arch == "amd64" {
					if fn := mod.NamedFunction("simd/archsimd.Uint32x4.SHA1Message1"); fn.IsNil() || strings.Contains(fn.String(), "PanicSIMDUnimplemented") {
						t.Fatal("missing callable SHA implementation")
					}
				}
				if target.arch == "arm64" {
					fn := mod.NamedFunction("simd/archsimd.Int8x16.LeadingZeros")
					if fn.IsNil() || !strings.Contains(fn.String(), "llvm.ctlz") {
						t.Fatal("missing callable leading zero count")
					}
				}
				if target.arch == "wasm" {
					fn := mod.NamedFunction("simd/archsimd.Int8x16.MulWidenHi")
					if fn.IsNil() || !strings.Contains(fn.String(), "shufflevector") || !strings.Contains(fn.String(), "<i32 8, i32 9, i32 10, i32 11, i32 12, i32 13, i32 14, i32 15>") {
						t.Fatal("missing callable high-half widening multiply")
					}
				}
				if fn := mod.NamedFunction("simd/archsimd.Uint8x16.ExtendLo8ToUint16"); fn.IsNil() || !strings.Contains(fn.String(), "zext <8 x i8>") {
					t.Fatal("integer extension has no callable vector definition")
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

func TestSIMDX86PackedInstructions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module packed\n\ngo 1.27\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const src = `package main
import "simd/archsimd"
//go:noinline
func sad(out *[8]uint64, a,b *[64]uint8) {
 archsimd.LoadUint8x64Array(a).SumOf8AbsDiff(archsimd.LoadUint8x64Array(b)).StoreArray(out)
}
//go:noinline
func sad32(out *[4]uint64, a,b *[32]uint8) {
 archsimd.LoadUint8x32Array(a).SumOf8AbsDiff(archsimd.LoadUint8x32Array(b)).StoreArray(out)
}
//go:noinline
func expandSum(out,a,b *[16]uint32, m uint16) {
 x := archsimd.LoadUint32x16Array(a).Add(archsimd.LoadUint32x16Array(b))
 x.Expand(archsimd.Mask32x16FromBits(m)).StoreArray(out)
}
//go:noinline
func expandFloat(out,a *[16]float32, m uint16) {
 archsimd.LoadFloat32x16Array(a).Expand(archsimd.Mask32x16FromBits(m)).StoreArray(out)
}
//go:noinline
func dot(out *[32]int16, a *[64]uint8, b *[64]int8) {
 archsimd.LoadUint8x64Array(a).DotProductPairsSaturated(archsimd.LoadInt8x64Array(b)).StoreArray(out)
}
//go:noinline
func pack(out *[16]uint32, a *[16]uint32, m uint16) {
 archsimd.LoadUint32x16Array(a).Compress(archsimd.Mask32x16FromBits(m)).StoreArray(out)
}
//go:noinline
func expand(out *[16]uint32, a *[16]uint32, m uint16) {
 archsimd.LoadUint32x16Array(a).Expand(archsimd.Mask32x16FromBits(m)).StoreArray(out)
}
//go:noinline
func floatPack(out, a *[4]float32, m uint8) {
 archsimd.LoadFloat32x4Array(a).Compress(archsimd.Mask32x4FromBits(m)).StoreArray(out)
}
//go:noinline
func fused(out,a,b,c *[16]float32) {
 x,y,z := archsimd.LoadFloat32x16Array(a),archsimd.LoadFloat32x16Array(b),archsimd.LoadFloat32x16Array(c)
 x.MulAddEvenSubOdd(y,z).StoreArray(out)
}
func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	for _, level := range []string{"v1", "v3", "v4"} {
		t.Run(level, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", level, "simd"
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			simdCodegenVariants(t, []string{"O0", "O2"}, pkgs[:1], func(t *testing.T, mod llvm.Module, asm string) {
				body := func(name string) string {
					t.Helper()
					match := regexp.MustCompile(`(?ms)^main\.` + name + `:[^\n]*\n(.*?)^\s*\.size\s`).FindStringSubmatch(asm)
					if match == nil {
						t.Fatalf("missing assembly for %s", name)
					}
					return match[1]
				}
				if level == "v1" {
					assertSIMDBaselineAssembly(t, mod, asm)
					if !regexp.MustCompile(`(?m)^\s*psadbw\s+[^\n]*%xmm`).MatchString(asm) {
						t.Error("missing baseline SSE2 SAD")
					}
					return
				}
				checkSAD := func(name, reg string, count int) {
					t.Helper()
					fn := body(name)
					wide := regexp.MustCompile(`vpsadbw\s+[^\n]*` + reg)
					if strings.Count(fn, "vpsadbw") != count || len(wide.FindAllString(fn, -1)) != count {
						t.Errorf("expected %d %s SAD in %s:\n%s", count, reg, name, fn)
					}
				}
				checkSAD("sad32", "%ymm", 1)
				if level == "v4" {
					checkSAD("sad", "%zmm", 1)
				} else {
					checkSAD("sad", "%ymm", 2)
				}
				if level != "v4" || !strings.HasSuffix(t.Name(), "/O2") {
					return
				}
				for _, tc := range []struct{ name, pattern string }{{"expandSum", `vpexpandd\s+%zmm`}, {"expandFloat", `vexpandps\s+\(%r\w+\),\s*%zmm`}} {
					fn := body(tc.name)
					if !regexp.MustCompile(tc.pattern).MatchString(fn) || strings.Contains(fn, "(%rsp)") || strings.Contains(fn, "(%rbp)") {
						t.Errorf("expected stack-free packed expand in %s:\n%s", tc.name, fn)
					}
				}
				for _, mnemonic := range []string{"vpmaddubsw", "vpcompressd", "vpexpandd", `vfmaddsub(132|213|231)ps`} {
					if !regexp.MustCompile(`(?m)^\s*` + mnemonic + `\s+[^\n]*%zmm`).MatchString(asm) {
						t.Errorf("missing 512-bit %s", mnemonic)
					}
				}
			})
		})
	}
}

func TestSIMDGuardedPackedInstructions(t *testing.T) {
	dir := t.TempDir()
	const src = `package main
import "simd/archsimd"
//go:noinline
func packed(o,a *[16]uint32, m uint16) {
 if archsimd.X86.AVX512() { archsimd.LoadUint32x16Array(a).Compress(archsimd.Mask32x16FromBits(m)).StoreArray(o) }
}
//go:noinline
func sad(o *[8]uint64, a,b *[64]uint8) {
 if archsimd.X86.AVX512() { archsimd.LoadUint8x64Array(a).SumOf8AbsDiff(archsimd.LoadUint8x64Array(b)).StoreArray(o) }
}
//go:noinline
func bytes(o,a *[64]uint8, m uint64) {
 if archsimd.X86.AVX512VBMI2() { archsimd.LoadUint8x64Array(a).Compress(archsimd.Mask8x64FromBits(m)).StoreArray(o) }
}
//go:noinline
func words(o,a *[32]uint16, m uint32) {
 if archsimd.X86.AVX512VBMI2() { archsimd.LoadUint16x32Array(a).Expand(archsimd.Mask16x32FromBits(m)).StoreArray(o) }
}
func main() {}
`
	for name, body := range map[string]string{"go.mod": "module guardedpacked\n\ngo 1.27\n", "main.go": src} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		level, guard string
		ops          map[string]string
	}{
		{"v1", "avx512", map[string]string{"packed": "vpcompressd", "sad": "vpsadbw"}},
		{"v4", "avx512vbmi2", map[string]string{"bytes": "vpcompressb", "words": "vpexpandw"}},
	} {
		t.Run(tc.level, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", tc.level, "simd"
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			simdCodegenVariants(t, []string{"O2"}, pkgs, func(t *testing.T, mod llvm.Module, asm string) {
				for fn, mnemonic := range tc.ops {
					for _, suffix := range []string{tc.guard, "base"} {
						symbol := "main." + fn + ".__llgo_fmv_guard_" + suffix
						body := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(symbol) + `:[^\n]*\n(.*?)^\s*\.size\s`).FindStringSubmatch(asm)
						if body == nil {
							t.Fatalf("missing assembly for %s", symbol)
						}
						packed := regexp.MustCompile(`(?m)^\s*` + mnemonic + `\s+[^\n]*%zmm`).MatchString(body[1])
						if packed != (suffix == tc.guard) {
							t.Errorf("%s: unexpected instruction selection for %s:\n%s", symbol, mnemonic, body[1])
						}
					}
				}
			})
		})
	}
}
