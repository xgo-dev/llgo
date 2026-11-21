package build

import (
	"path/filepath"
	"runtime"
	"slices"
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
			if bucket.TypeKind() != llvm.StructTypeKind {
				t.Fatalf("memory profile bucket is not a struct: %s", bucket.String())
			}
			fields := bucket.StructElementTypes()
			if len(fields) != 1 || fields[0].TypeKind() != llvm.ArrayTypeKind {
				t.Fatalf("memory profile bucket has no counter shard array: %s", bucket.String())
			}
			shard := fields[0].ElementType()
			if shard.TypeKind() != llvm.StructTypeKind {
				t.Fatalf("memory profile shard is not a struct: %s", shard.String())
			}
			index := slices.IndexFunc(shard.StructElementTypes(), func(field llvm.Type) bool {
				return field.TypeKind() == llvm.IntegerTypeKind && field.IntTypeWidth() == 64
			})
			if index < 0 {
				t.Fatalf("memory profile shard has no 64-bit counter: %s", shard.String())
			}
			offset := td.ElementOffset(bucket, 0) + td.ElementOffset(shard, index)
			stride := td.TypeAllocSize(bucket)
			shardStride := td.TypeAllocSize(shard)
			if buckets.Alignment() < 8 || offset%8 != 0 || stride%8 != 0 || shardStride%8 != 0 {
				t.Fatalf("unaligned 64-bit atomic counters: base alignment=%d, field offset=%d, bucket stride=%d, shard stride=%d",
					buckets.Alignment(), offset, stride, shardStride)
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
