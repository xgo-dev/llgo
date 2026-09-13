//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

const simd128Source = `package main
import "simd/archsimd"

//go:noinline
func add(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Add(y) }
//go:noinline
func lane(x archsimd.Float32x4, i uint8) float32 { return x.GetElem(i) }
func main() {
 var x, y archsimd.Float32x4
 x = x.SetElem(0, 1.5).SetElem(3, -2)
 y = y.SetElem(0, 2.5).SetElem(3, 5)
 z := add(x, y)
 if lane(z, 0) != 4 || lane(z, 1) != 0 || lane(z, 3) != 3 { panic("float add") }
 sub := z.Sub
 z = sub(y)
 if lane(z, 0) != 1.5 || lane(z, 3) != -2 { panic("method value") }
 var a, b archsimd.Uint64x2
 a = a.SetElem(0, ^uint64(0)).SetElem(1, 7)
 b = b.SetElem(0, 1).SetElem(1, 3)
 c := a.Add(b)
 if c.GetElem(0) != 0 || c.GetElem(1) != 10 { panic("integer wrap") }
 c = a.Xor(b)
 if c.GetElem(0) != ^uint64(1) || c.GetElem(1) != 4 { panic("xor") }
 bad := func() (ok bool) {
  defer func() { if e := recover(); e != nil { ok = e.(interface{Error() string}).Error() == "runtime error: out-of-range immediate for simd intrinsic" } }()
  lane(z, 4)
  return
 }
 if !bad() { panic("lane bounds") }
 badDeferred := func() (ok bool) {
  defer func() { ok = recover() != nil }()
  defer z.GetElem(255)
  return
 }
 if !badDeferred() { panic("deferred lane bounds") }
 c = a.And(b).Or(b)
 if c.GetElem(0) != 1 || c.GetElem(1) != 3 { panic("and/or") }
 var small archsimd.Int8x16
 for i := uint8(0); i < 16; i++ { small = small.SetElem(i, int8(i)+120) }
 small = small.Add(small)
 for i := uint8(0); i < 16; i++ { want := int8(i)+120; if small.GetElem(i) != want+want { panic("int8 lanes") } }
 var d archsimd.Float64x2
 d = d.SetElem(0, -1.25).SetElem(1, 2.5)
 d = d.Add(d)
 if d.GetElem(0) != -2.5 || d.GetElem(1) != 5 { panic("float64 lanes") }
}
`

func simdTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{"go.mod": "module simdtest\n\ngo 1.27\n", "main.go": simd128Source} {
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

		})
	}
}

func TestSIMD128Execution(t *testing.T) {
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		t.Skip("native SIMD128 execution")
	}
	dir := simdTestDir(t)
	native := exec.Command("go", "run", ".")
	native.Dir = dir
	native.Env = withEnv(os.Environ(), "GOEXPERIMENT=simd")
	if out, err := native.CombinedOutput(); err != nil {
		t.Fatalf("official Go: %v\n%s", err, out)
	}
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(level.Flag(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.GOEXPERIMENT = "simd"
			conf.OptLevel = level
			conf.OutFile = filepath.Join(dir, "simd-test")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(conf.OutFile).CombinedOutput(); err != nil {
				t.Fatalf("LLGo: %v\n%s", err, out)
			}
		})
	}

}

func TestSIMD128WASIExecution(t *testing.T) {
	wasmtime, err := exec.LookPath("wasmtime")
	if err != nil {
		t.Skip("wasmtime is required for WASI execution")
	}
	wasmOpt := os.Getenv("WASMOPT")
	if wasmOpt == "" {
		wasmOpt = "wasm-opt"
	}
	if _, err := exec.LookPath(wasmOpt); err != nil {
		t.Skip("wasm-opt is required for WASI Asyncify")
	}
	dir := simdTestDir(t)
	nativeWasm := filepath.Join(dir, "official.wasm")
	native := exec.Command("go", "build", "-o", nativeWasm, ".")
	native.Dir = dir
	native.Env = withEnv(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=simd")
	if out, err := native.CombinedOutput(); err != nil {
		t.Fatalf("official Go wasm: %v\n%s", err, out)
	}
	if out, err := exec.Command(wasmtime, nativeWasm).CombinedOutput(); err != nil {
		t.Fatalf("official Go WASI: %v\n%s", err, out)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.Target, conf.GOEXPERIMENT = "wasi", "simd"
	conf.OutFile = filepath.Join(dir, "simd-test.wasm")
	if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(wasmtime, conf.OutFile).CombinedOutput(); err != nil {
		t.Fatalf("WASI: %v\n%s", err, out)
	}
}
