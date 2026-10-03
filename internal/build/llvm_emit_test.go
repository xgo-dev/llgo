package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWASILTOFunctionFeatures(t *testing.T) {
	for _, test := range []struct {
		name    string
		profile crosscompile.WasmProfile
		mode    lto.Mode
		want    bool
	}{
		{"WASI full", crosscompile.WasmProfileW32, lto.Full, true},
		{"WASI thin", crosscompile.WasmProfileW32, lto.Thin, true},
		{"WASI no LTO", crosscompile.WasmProfileW32, lto.Off, false},
		{"browser", crosscompile.WasmProfileJ32, lto.Full, false},
		{"native", crosscompile.WasmProfileNone, lto.Full, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			lc := llvm.NewContext()
			defer lc.Dispose()
			mod := lc.NewModule("features")
			defer mod.Dispose()
			fnType := llvm.FunctionType(lc.VoidType(), nil, false)
			fn := llvm.AddFunction(mod, "function", fnType)
			builder := lc.NewBuilder()
			builder.SetInsertPointAtEnd(lc.AddBasicBlock(fn, "entry"))
			builder.CreateRetVoid()
			plain := llvm.AddFunction(mod, "plain", fnType)
			builder.SetInsertPointAtEnd(lc.AddBasicBlock(plain, "entry"))
			builder.CreateRetVoid()
			builder.Dispose()
			fn.AddFunctionAttr(lc.CreateStringAttribute("target-features", "+simd128,-atomics,-exception-handling"))
			llvm.AddFunction(mod, "declaration", fnType)
			ctx := &context{buildConf: &Config{LTO: test.mode}, crossCompile: crosscompile.Export{WasmProfile: test.profile}}
			applyWASILTOFeatures(ctx, mod)
			ir := mod.String()
			for _, feature := range []string{"+atomics", "+bulk-memory", "+exception-handling"} {
				if strings.Contains(ir, feature) != test.want {
					t.Fatalf("feature %q, want present=%v:\n%s", feature, test.want, ir)
				}
			}
			if !strings.Contains(ir, "+simd128") || strings.Contains(ir, "declare void @declaration() #") {
				t.Fatalf("existing features or declaration changed:\n%s", ir)
			}
			if test.want && (strings.Contains(ir, "-atomics") || strings.Contains(ir, "-exception-handling")) {
				t.Fatalf("required features still have negative counterparts:\n%s", ir)
			}
			applyWASILTOFeatures(ctx, mod)
			if mod.String() != ir {
				t.Fatal("applying LTO features twice changed the module")
			}
		})
	}
}

func TestUseInMemoryNativeCodegenConf(t *testing.T) {
	t.Run("native host", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH}
		if !useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected native host build to use in-memory native codegen")
		}
	})

	t.Run("embedded target", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, Target: "rp2040"}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected embedded target build to keep using clang")
		}
	})

	t.Run("full LTO", func(t *testing.T) {
		conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, LTO: lto.Full}
		if !useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected native full LTO build to use in-memory bitcode emission")
		}
	})

	t.Run("cross compile host mismatch", func(t *testing.T) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		if goos == "linux" {
			goos = "darwin"
		} else {
			goos = "linux"
		}
		if goarch == "amd64" {
			goarch = "arm64"
		} else {
			goarch = "amd64"
		}
		conf := &Config{Goos: goos, Goarch: goarch}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected host mismatch to keep using clang")
		}
	})

	t.Run("wasm", func(t *testing.T) {
		conf := &Config{Goos: "wasip1", Goarch: "wasm"}
		if useInMemoryNativeCodegenConf(conf) {
			t.Fatal("expected wasm target to keep using clang")
		}
	})
}

func TestExportPackageObjectErrors(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("p", "example.com/p")

	t.Run("clang", func(t *testing.T) {
		ctx := &context{
			buildConf: &Config{Target: "embedded"},
			crossCompile: crosscompile.Export{
				CC: filepath.Join(t.TempDir(), "missing-clang"),
			},
			commands: commandEnv{environ: os.Environ()},
		}
		path, member, err := exportPackageObject(ctx, pkg.Path(), "p.o", pkg)
		if path != "" {
			defer os.Remove(path)
		}
		if err == nil {
			member.buffer.Dispose()
			t.Fatal("exportPackageObject succeeded with a missing clang")
		}
		if !member.buffer.IsNil() {
			member.buffer.Dispose()
			t.Fatal("clang export returned an in-memory archive member")
		}
	})

	t.Run("IR dump", func(t *testing.T) {
		ctx := &context{buildConf: &Config{
			Goos:         runtime.GOOS,
			Goarch:       runtime.GOARCH,
			CheckLLFiles: true,
		}}
		exportFile := strings.Repeat("x", 300)
		path, member, err := exportPackageObject(ctx, pkg.Path(), exportFile, pkg)
		if path != "" {
			defer os.Remove(path)
		}
		if err == nil {
			member.buffer.Dispose()
			t.Fatal("exportPackageObject succeeded with an overlong IR dump prefix")
		}
		if !member.buffer.IsNil() {
			member.buffer.Dispose()
			t.Fatal("failed IR dump returned an in-memory archive member")
		}
	})
}
