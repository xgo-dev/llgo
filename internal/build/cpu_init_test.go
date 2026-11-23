//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// This fixture deliberately has no public runtime import. CPU detection must
// also work when archsimd is the only standard-library dependency.
const cpuInitAMD64Source = `package main
import "simd/archsimd"
var earlyAVX = archsimd.X86.AVX()
var earlyAVX2 = archsimd.X86.AVX2()
func main() {
 println(earlyAVX, earlyAVX2, archsimd.X86.AVX(), archsimd.X86.AVX2(), archsimd.X86.FMA(), archsimd.X86.AVXAES())
}

`

const cpuInitARM64Source = `package main
import _ "simd/archsimd"
import _ "unsafe"
// Match the official internal/cpu.ARM64 prefix. That variable explicitly
// supports linkname users; no private initialization function is called here.
//go:linkname flags internal/cpu.ARM64
var flags struct {
 _ [128]byte
 AES, PMULL, SHA1, SHA2, SHA512, SHA3, CRC32, ATOMICS, CPUID, DIT, SB, Neoverse bool
 _ [128]byte
}
var earlyAES = flags.AES
func main() {
 println(earlyAES, flags.AES, flags.PMULL, flags.SHA1, flags.SHA2, flags.CRC32)
}
`

func TestCPUInitializationMatchesGo(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("CPU fixture requires amd64 or arm64")
	}
	dir := t.TempDir()
	source := cpuInitAMD64Source
	if runtime.GOARCH == "arm64" {
		source = cpuInitARM64Source
	}
	for name, text := range map[string]string{"go.mod": "module cpuprobe\n\ngo 1.27\n", "main.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	reference := filepath.Join(dir, "official")
	if runtime.GOOS == "windows" {
		reference += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", reference, ".")
	cmd.Dir = dir
	cmd.Env = withEnv(os.Environ(), "GOEXPERIMENT=simd", "GOAMD64=v1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("official build: %v\n%s", err, out)
	}
	run := func(bin, debug string) string {
		t.Helper()
		cmd := exec.Command(bin)
		cmd.Env = withEnv(os.Environ(), "GODEBUG="+debug)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run with GODEBUG=%q: %v\n%s", debug, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	debugOptions := []string{"", "cpu.all=off", "cpu.aes=off", "cpu.all=off,cpu.aes=on", "cpu.avx=off"}
	if runtime.GOARCH == "arm64" {
		debugOptions = debugOptions[:4]
	}
	// Go 1.27 initializes CPU flags before reading Windows environment
	// variables. Compare default detection there, then check LLGo overrides
	// against explicit expectations instead of that reference limitation.
	baseline := strings.Fields(run(reference, ""))
	if len(baseline) != 6 {
		t.Fatalf("unexpected CPU output: %v", baseline)
	}
	expected := func(debug string) string {
		if debug == "" {
			return strings.Join(baseline, " ")
		}
		values := append([]string(nil), baseline...)
		switch debug {
		case "cpu.all=off", "cpu.all=off,cpu.aes=on":
			for i := range values {
				values[i] = "false"
			}
			if debug == "cpu.all=off,cpu.aes=on" && runtime.GOARCH == "arm64" {
				values[0], values[1] = baseline[0], baseline[1]
			}
		case "cpu.aes=off":
			if runtime.GOARCH == "arm64" {
				values[0], values[1] = "false", "false"
			} else {
				values[5] = "false"
			}
		case "cpu.avx=off":
			values[0], values[2], values[5] = "false", "false", "false"
		default:
			t.Fatalf("missing expectation for GODEBUG=%q", debug)
		}
		return strings.Join(values, " ")
	}
	for _, mode := range []struct {
		name  string
		level optlevel.Level
		lto   lto.Mode
	}{
		{"O0", optlevel.O0, lto.Off}, {"O2", optlevel.O2, lto.Off},
		{"O2-thin", optlevel.O2, lto.Thin}, {"O2-full", optlevel.O2, lto.Full},
	} {
		t.Run(mode.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.GOEXPERIMENT, conf.OptLevel, conf.LTO = "simd", mode.level, mode.lto
			if runtime.GOARCH == "amd64" {
				conf.GOAMD64 = "v1"
			}
			conf.OutFile = filepath.Join(dir, "llgo-"+mode.name)
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			for _, debug := range debugOptions {
				got := run(conf.OutFile, debug)
				if want := expected(debug); got != want {
					t.Fatalf("GODEBUG=%q: LLGo %q, expected %q", debug, got, want)
				}
				if runtime.GOOS != "windows" {
					if want := run(reference, debug); got != want {
						t.Fatalf("GODEBUG=%q: LLGo %q, official Go %q", debug, got, want)
					}
				}
			}
		})
	}
}

func TestCPUInitializationTargetHooks(t *testing.T) {
	for _, target := range []struct{ os, arch string }{
		{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"}, {"wasip1", "wasm"},
	} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch = target.os, target.arch
			pkgs, err := Do([]string{"internal/cpu"}, conf)
			if err != nil {
				t.Fatal(err)
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			initialized := false
			for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
				if strings.HasPrefix(fn.Name(), "internal/cpu.init") && strings.Contains(fn.String(), "internal/cpu.Initialize") {
					initialized = strings.Contains(fn.String(), "CPUEnvironment")
				}
			}
			if target.arch == "wasm" {
				// The Wasm detector has no CPU options. Keep host environment
				// reads out of its package initializer.
				if initialized {
					t.Fatal("Wasm CPU initialization reads process overrides")
				}
			} else if !initialized {
				t.Fatal("CPU initialization does not apply process overrides")
			}
			var bridge string
			switch target.os {
			case "darwin":
				bridge = "internal/cpu.sysctlbynameInt32"
			case "windows":
				bridge = "internal/cpu.isProcessorFeaturePresent"
			case "linux":
				if target.arch == "arm64" {
					if !strings.Contains(mod.NamedFunction("internal/cpu.llgoPrepareCPU").String(), "@getauxval") {
						t.Fatal("missing AT_HWCAP initialization")
					}
				}
			}
			if bridge != "" {
				fn := mod.NamedFunction(bridge)
				if fn.IsNil() || fn.IsDeclaration() {
					t.Fatalf("CPU bridge %s depends on public runtime", bridge)
				}
			}
		})
	}
}
