package build

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestMemProfileCounterAlignment(t *testing.T) {
	llvmgen.Initialize(llvmgen.InitAll)
	root, err := filepath.Abs("../../..")
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
			targetInfo := &llvmgen.Target{GOOS: conf.Goos, GOARCH: conf.Goarch}
			if _, err := llvm.GetTargetFromTriple(targetInfo.Spec().Triple); err != nil {
				// The official Windows LLVM SDK omits the MIPS backend.
				// Keep the core 386/ARM targets mandatory on every host.
				if runtime.GOOS == "windows" && conf.Goarch == "mips" {
					t.Skipf("Windows LLVM build lacks MIPS: %v", err)
				}
				t.Fatal(err)
			}
			pkgs, err := Build(Invocation{Args: []string{llvmgen.PkgRuntime}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			shards := mod.NamedGlobal(llvmgen.PkgRuntime + ".memProfileShardsData")
			if shards.IsNil() {
				t.Fatal("missing memory profile shards")
			}
			td := llvm.NewTargetData(mod.DataLayout())
			defer td.Dispose()
			array := shards.GlobalValueType()
			if array.TypeKind() != llvm.ArrayTypeKind {
				t.Fatalf("memory profile shards are not an array: %s", array.String())
			}
			shard := array.ElementType()
			offset := td.ElementOffset(shard, 1)
			stride := td.TypeAllocSize(shard)
			if shards.Alignment() < 8 || offset%8 != 0 || stride%8 != 0 {
				t.Fatalf("unaligned 64-bit atomic counters: base alignment=%d, field offset=%d, stride=%d", shards.Alignment(), offset, stride)
			}
			if stride != 256 {
				t.Fatalf("unexpected shard stride: got %d, want 256", stride)
			}
			// Allocator accounting must remain an intrinsic at O0 too: an
			// instrumented stdlib method can allocate and re-enter profiling.
			add := mod.NamedFunction(llvmgen.PkgRuntime + ".memProfileAddBucket")
			if add.IsNil() || !strings.Contains(add.String(), "atomicrmw add") || !strings.Contains(add.String(), "monotonic") {
				t.Fatal("memory profile updates must use the allocation-free relaxed runtime intrinsic")
			}
		})
	}
}
