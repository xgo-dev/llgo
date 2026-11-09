package build

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestMemProfileCounterAlignment(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, target := range []string{
		"windows/386", "linux/386", "linux/arm", "linux/mips",
	} {
		t.Run(target, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, _ = strings.Cut(target, "/")
			targetInfo := &llssa.Target{GOOS: conf.Goos, GOARCH: conf.Goarch}
			if _, err := llvm.GetTargetFromTriple(targetInfo.Spec().Triple); err != nil {
				// The official Windows LLVM SDK omits the MIPS backend.
				// Keep the core 386/ARM targets mandatory on every host.
				if runtime.GOOS == "windows" && conf.Goarch == "mips" {
					t.Skipf("Windows LLVM build lacks MIPS: %v", err)
				}
				t.Fatal(err)
			}
			pkgs, err := Build(Invocation{Args: []string{llssa.PkgRuntime}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			buckets := mod.NamedGlobal(llssa.PkgRuntime + ".memProfileBuckets")
			if buckets.IsNil() {
				t.Fatal("missing memory profile buckets")
			}
			td := llvm.NewTargetData(mod.DataLayout())
			defer td.Dispose()
			array := buckets.GlobalValueType()
			if array.TypeKind() != llvm.ArrayTypeKind {
				t.Fatalf("memory profile buckets are not an array: %s", array.String())
			}
			bucket := array.ElementType()
			offset := td.ElementOffset(bucket, 1)
			stride := td.TypeAllocSize(bucket)
			if buckets.Alignment() < 8 || offset%8 != 0 || stride%8 != 0 {
				t.Fatalf("unaligned 64-bit atomic counters: base alignment=%d, field offset=%d, stride=%d", buckets.Alignment(), offset, stride)
			}
			// Allocator accounting must remain an intrinsic at O0 too: an
			// instrumented stdlib method can allocate and re-enter profiling.
			add := mod.NamedFunction(llssa.PkgRuntime + ".memProfileAddObject")
			if add.IsNil() || !strings.Contains(add.String(), "atomicrmw add") {
				t.Fatal("memory profile updates must use the allocation-free runtime intrinsic")
			}
		})
	}
}
