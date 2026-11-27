package build

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/crosscompile"
	"github.com/xgo-dev/llgo/cmd/compiler/meta"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func dispatchInlineContext() *context {
	return &context{passOpt: true, conf: &packages.Config{}, buildConf: dispatchInlineConfig(), crossCompile: crosscompile.Export{
		LLVMTarget: "x86_64-unknown-linux-gnu",
		Toolchain:  crosscompile.NativeToolchain{ObjectFormat: crosscompile.ObjectFormatELF, Linker: crosscompile.LinkerFlavorELFLLD},
	}}
}

func TestDispatchInlineBackendOptionParsing(t *testing.T) {
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		t.Setenv(name, "")
	}
	for _, tt := range []struct {
		flags          []string
		inline, layout bool
	}{
		{[]string{"-L/opt/inline-libs", "-I/opt/align-loops", "-finline-functions", "-fno-inline-small-functions"}, true, true},
		{[]string{"-mllvm", "-inline-threshold=20"}, false, true},
		{[]string{"--mllvm", "-inline-threshold=20"}, false, true},
		{[]string{"-mllvm=-inlinehint-threshold=20"}, false, true},
		{[]string{"--mllvm=-inlinehint-threshold=20"}, false, true},
		{[]string{"-Xlinker", "-mllvm", "-Xlinker", "-cold-callsite-rel-freq=0"}, false, true},
		{[]string{"-Xlinker", "--mllvm", "-Xlinker", "-cold-callsite-rel-freq=0"}, false, true},
		{[]string{"-Wl,-mllvm,-align-loops=16,-mllvm,-inline-threshold=20"}, false, false},
		{[]string{"-Wl,--mllvm,-align-loops=16,--mllvm,-inline-threshold=20"}, false, false},
		{[]string{"-Wl,--mllvm,-inline-threshold=20"}, false, true},
		{[]string{"-Wl,--mllvm=-inline-threshold=20"}, false, true},
		{[]string{"-Wl,-plugin-opt=-inline-threshold=20"}, false, true},
		{[]string{"-Wl,--plugin-opt=-inline-threshold=20"}, false, true},
		{[]string{"-Wl,-plugin-opt=-align-loops=16"}, true, false},
		{[]string{"-Wl,--plugin-opt=-align-loops=16"}, true, false},
		{[]string{"-Wl,-plugin-opt=O2,--plugin-opt=jobs=4"}, true, true},
		{[]string{"-Wl,-plugin-opt=inline-threshold=20"}, true, true},
		{[]string{"-Wl,-mllvm=-x86-align-branch-boundary=64"}, true, false},
		{[]string{"-Wl,--mllvm=-x86-align-branch-boundary=64"}, true, false},
	} {
		c := dispatchInlineContext()
		c.crossCompile.CXXArgs = tt.flags
		if got := c.dispatchInlineBudgetEnabled(); got != tt.inline {
			t.Errorf("%v: inline=%v, want %v", tt.flags, got, tt.inline)
		}
		got := slices.Contains(c.nativeFullLTOLinkerArgs(), "-Wl,-mllvm,-align-loops=64")
		if got != tt.layout {
			t.Errorf("%v: layout=%v, want %v", tt.flags, got, tt.layout)
		}
	}
	for _, flags := range []string{
		"-Wl,-mllvm,-inline-threshold=20",
		"-Wl,--mllvm,-inline-threshold=20",
		"-Wl,--mllvm=-inline-threshold=20",
		"-Wl,-plugin-opt=-inline-threshold=20",
		"-Wl,--plugin-opt=-inline-threshold=20",
	} {
		c := dispatchInlineContext()
		c.commands.environ = []string{"LDFLAGS=" + flags}
		if c.dispatchInlineBudgetEnabled() {
			t.Errorf("ignored captured command environment: %s", flags)
		}
	}
}

func TestDispatchInlineArchiveCache(t *testing.T) {
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS", "LLGO_BUILD_CACHE"} {
		t.Setenv(name, "")
	}
	root := t.TempDir()
	previous := cacheRootFunc
	cacheRootFunc = func() string { return root }
	t.Cleanup(func() { cacheRootFunc = previous })
	archive := filepath.Join(root, "input.a")
	if err := os.WriteFile(archive, []byte("!<arch>\n"), 0644); err != nil {
		t.Fatal(err)
	}
	seen := map[bool]string{}
	for _, enabled := range []bool{true, false, true, false} {
		flags := "-Wl,-mllvm,-inline-threshold=20"
		if enabled {
			flags = ""
		}
		t.Setenv("LDFLAGS", flags)
		c := dispatchInlineContext()
		m := newManifestBuilder()
		c.collectCommonInputs(m)
		if m.common.DispatchInlineBudget != enabled {
			t.Fatalf("fingerprint policy=%v, want %v", m.common.DispatchInlineBudget, enabled)
		}
		pkg := &aPackage{Package: &packages.Package{ID: "dep", PkgPath: "example.test/dep", Name: "dep"},
			Fingerprint: m.Fingerprint(), Manifest: m.Build()}
		if prior, warm := seen[enabled]; warm {
			if pkg.Fingerprint != prior || !c.tryLoadFromCache(pkg) {
				t.Fatal("did not restore the matching policy archive")
			}
		} else {
			if c.tryLoadFromCache(pkg) {
				t.Fatal("loaded an archive compiled with a different inline policy")
			}
			mod := parseWasmAggregateIR(t, dispatchInlineIR)
			applyPackageDispatchInlineBudget(c, pkg, mod)
			pkg.ArchiveFile = archive
			pkg.Meta, _ = meta.NewBuilder().Build()
			if err := c.saveToCache(pkg); err != nil {
				t.Fatal(err)
			}
			if pkg.Meta != nil {
				pkg.Meta.Close()
			}
			seen[enabled] = pkg.Fingerprint
		}
		if got := slices.Contains(pkg.LinkArgs, dispatchInlineCostFlag); got != enabled {
			t.Fatalf("cached complete-cost flag=%v, want %v", got, enabled)
		}
		if pkg.CacheHit && pkg.Meta != nil {
			pkg.Meta.Close()
		}
	}
	if seen[true] == seen[false] {
		t.Fatal("changing a link-only environment override reused hinted bitcode")
	}
}

func TestDispatchInlineCompleteCostRequiresHints(t *testing.T) {
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		t.Setenv(name, "")
	}
	profile := dispatchInlineIR + `
!llvm.module.flags = !{!0}
!0 = !{i32 1, !"ProfileSummary", !1}
!1 = !{!2, !3, !4, !5, !6, !7, !8, !9}
!2 = !{!"ProfileFormat", !"InstrProf"}
!3 = !{!"TotalCount", i64 5}
!4 = !{!"MaxCount", i64 5}
!5 = !{!"MaxInternalCount", i64 5}
!6 = !{!"MaxFunctionCount", i64 5}
!7 = !{!"NumCounts", i64 1}
!8 = !{!"NumFunctions", i64 1}
!9 = !{!"DetailedSummary", !10}
!10 = !{!11}
!11 = !{i32 999999, i64 5, i32 1}
`
	for _, source := range []string{profile, `define void @plain() { ret void }`,
		strings.Replace(dispatchInlineIR, "br label %switch", "br label %exit", 1)} {
		pkg := &aPackage{}
		applyPackageDispatchInlineBudget(dispatchInlineContext(), pkg, parseWasmAggregateIR(t, source))
		if len(pkg.LinkArgs) != 0 {
			t.Fatalf("complete-cost flag without eligible calls: %v", pkg.LinkArgs)
		}
	}
	args := []string{"-lfoo", dispatchInlineCostFlag, "-lbar", dispatchInlineCostFlag}
	if got := deduplicateDispatchInlineCost(args); !slices.Equal(got, []string{"-lfoo", dispatchInlineCostFlag, "-lbar"}) {
		t.Fatalf("linked-package cost options = %v", got)
	}
}
