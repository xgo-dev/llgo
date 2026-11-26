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

func TestSIMDMidwayLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module midwaytest\n\ngo 1.27\n",
				"main.go": `package main
import "simd"
type Block struct { V simd.Uint32s; N int }
var global = makeBlock(3)
func makeBlock(x uint32) Block { return Block{simd.BroadcastUint32s(x), 7} }
//go:noinline
func sum(xs []uint32) uint32 {
 v:=global.V
 for len(xs)>=v.Len() { v=v.Add(simd.LoadUint32s(xs)); xs=xs[v.Len():] }
 out:=make([]uint32,v.Len());v.Store(out)
 var s uint32;for _,x:=range out { s+=x };return s
}
//export width
func width() int { return (simd.Uint32s{}).Len() }
func main() { println(sum([]uint32{1,2,3,4}),width()) }
`,
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			if target.arch == "amd64" {
				conf.GOAMD64 = "v1"
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, compiled := range pkgs {
				pkg := compiled.LPkg
				defer pkg.Prog.Dispose()
				mod := pkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if mod.NamedFunction("main.sum").IsNil() {
					continue
				}
				found = true
				for _, name := range []string{"main.sum@simd0", "main.sum@simd128", "main.makeBlock@simd0", "main.makeBlock@simd128"} {
					if fn := mod.NamedFunction(name); fn.IsNil() || fn.BasicBlocksCount() == 0 {
						t.Fatalf("missing variant %s", name)
					}
				}
				for _, name := range []string{"main.sum@simd256", "main.makeBlock@simd256", "main.sum@simd512", "main.makeBlock@simd512"} {
					fn := mod.NamedFunction(name)
					if target.arch == "amd64" {
						if fn.IsNil() || fn.BasicBlocksCount() == 0 {
							t.Fatalf("missing variant %s", name)
						}
					} else if !fn.IsNil() {
						t.Fatalf("unexpected wide variant %s on %s", name, target.arch)
					}
				}
				if !strings.Contains(mod.NamedFunction("main.sum").String(), "simd.Emulated") {
					t.Fatal("scalar boundary lost runtime selection")
				}
				if target.arch == "amd64" {
					attr := mod.NamedFunction("main.sum").GetStringAttributeAtIndex(-1, "target-features")
					if !attr.IsNil() && strings.Contains(attr.GetStringValue(), "+avx") {
						t.Fatalf("portable dispatcher requires %s at GOAMD64=v1", attr.GetStringValue())
					}
				}
				mod.SetTarget(pkg.Prog.Target().Spec().Triple)
				mod.SetDataLayout(pkg.Prog.DataLayout())
				for _, level := range []string{"O0", "O2"} {
					if level == "O2" {
						opts := llvm.NewPassBuilderOptions()
						opts.SetVerifyEach(true)
						err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), opts)
						opts.Dispose()
						if err != nil {
							t.Fatal(err)
						}
					}
					asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
					if err != nil {
						t.Fatal(err)
					}
					if target.os == "linux" && target.arch == "amd64" {
						code := string(asm.Bytes())
						assertSIMDBaselineAssembly(t, mod, code)
						if level == "O2" {
							for _, width := range []struct{ name, register string }{{"256", "%ymm"}, {"512", "%zmm"}} {
								body := regexp.MustCompile(`(?ms)^"main\.sum@simd` + width.name + `":[^\n]*\n(.*?)^\s*\.size\s`).FindStringSubmatch(code)
								if body == nil || !regexp.MustCompile(`(?m)^\s*vpaddd\s+[^\n]*`+width.register).MatchString(body[1]) {
									t.Errorf("main.sum@simd%s must use %s for packed addition", width.name, width.register)
								}
							}
						}
					}
					asm.Dispose()
				}
			}
			if !found {
				t.Fatal("main package was not compiled")
			}
		})
	}
}

func TestSIMDMidwayRejectGlobalBinding(t *testing.T) {
	for _, tc := range []struct{ name, decl, want string }{
		{"vector", `var V = simd.BroadcastUint32s(7)
//go:linkname Alias main.V
var Alias simd.Uint32s
func main() { out:=make([]uint32,4);Alias.Store(out);println(out[0]) }
`, "binding for SIMD-dependent global Alias"},
		{"named scalar", `type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
var V Number=7
//go:linkname Alias main.V
var Alias Number
func main() {println(Alias,V)}
`, "binding for SIMD-dependent global Alias"},
		{"scalar alias to dependent target", `type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
var V Number=7
//go:linkname Alias main.V
var Alias int
func main() {println(Alias,V)}
`, "target SIMD-dependent global main.V"},
		{"llgo link", `type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
var V Number=7
//llgo:link Alias main.V
var Alias Number
func main() {println(Alias,V)}
`, "llgo:link binding for SIMD-dependent global Alias"},
		{"explicit Go namespace", `type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
var V Number=7
//go:linkname Alias go:main.V
var Alias int
func main() {println(Alias,V)}
`, "target SIMD-dependent global go:main.V"},
		{"exported variable", `type Number int
func (Number) Width() int {var v simd.Uint32s;return v.Len()}
//export V
var V Number=7
func main() {println(V)}
`, "export binding for SIMD-dependent global V"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{"go.mod": "module midwaylinkname\n\ngo 1.27\n", "main.go": "package main\nimport (\"simd\"; _ \"unsafe\")\n" + tc.decl} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = "wasip1", "wasm", "simd"
			output := captureStderr(t)
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			diagnostic := output()
			if err == nil || !strings.Contains(diagnostic, tc.want) {
				t.Fatalf("got %v (%s), want %s", err, diagnostic, tc.want)
			}
		})
	}
}
