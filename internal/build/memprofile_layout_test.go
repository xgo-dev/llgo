package build

import (
	"path/filepath"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestMemProfileCounterAlignment(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, goos := range []string{"windows", "linux"} {
		t.Run(goos+"/386", func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch = goos, "386"
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
			bucket := buckets.GlobalValueType().ElementType()
			offset := td.ElementOffset(bucket, 1)
			stride := td.TypeAllocSize(bucket)
			if buckets.Alignment() < 8 || offset%8 != 0 || stride%8 != 0 {
				t.Fatalf("unaligned 64-bit atomic counters: base alignment=%d, field offset=%d, stride=%d", buckets.Alignment(), offset, stride)
			}
		})
	}
}
