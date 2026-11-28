package build

import (
	"github.com/xgo-dev/llgo/types"
	"go/token"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"weak"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func TestPlan9AsmSignatureCacheDoesNotRetainBuildContext(t *testing.T) {
	ref := func() weak.Pointer[context] {
		ctx := &context{plan9asmReady: true, plan9asmMode: plan9asmEnvNone}
		if _, err := plan9asmSigsForPkg(ctx, "example.com/disabled"); err != nil {
			t.Fatal(err)
		}
		return weak.Make(ctx)
	}()
	runtime.GC()
	if ref.Value() != nil {
		t.Fatal("Plan 9 signature cache retains a completed build context")
	}
}

func TestPlan9AsmSignatureCacheEmptyResults(t *testing.T) {
	if got, err := plan9asmSigsForPkg(nil, "p"); err != nil || got != nil {
		t.Fatalf("nil context: %v, %v", got, err)
	}
	ctx := &context{plan9asmReady: true, plan9asmMode: plan9asmEnvAll, buildConf: &Config{Goos: "linux", Goarch: "amd64"}}
	for _, path := range []string{"", "runtime", "example.com/missing"} {
		for range 2 {
			if got, err := plan9asmSigsForPkg(ctx, path); err != nil || len(got) != 0 {
				t.Fatalf("%s: %v, %v", path, got, err)
			}
		}
	}
}

func TestPlan9AsmSignatureCacheRetriesErrorsAndReusesSuccess(t *testing.T) {
	ctx, path, asm := newPlan9AsmSignatureTestContext(t)
	if _, err := plan9asmSigsForPkg(ctx, path); err == nil {
		t.Fatal("missing assembly source unexpectedly succeeded")
	}
	ctx.buildConf.Overlay[asm] = []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\nRET\n")
	got, err := plan9asmSigsForPkg(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[path+".Foo"]; !ok {
		t.Fatalf("missing Foo signature: %v", got)
	}
	// A successful lookup is cached; a failed one must not have been.
	delete(ctx.buildConf.Overlay, asm)
	if cached, err := plan9asmSigsForPkg(ctx, path); err != nil || len(cached) != 1 {
		t.Fatalf("cached translated signatures: %v, %v", cached, err)
	}
}

func TestPlan9AsmSignatureCacheIsPerContext(t *testing.T) {
	first, path, asm := newPlan9AsmSignatureTestContext(t)
	first.buildConf.Overlay[asm] = []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\nRET\n")
	second := &context{plan9asmReady: true, plan9asmMode: plan9asmEnvNone}
	load := func(ctx *context) map[string]struct{} {
		t.Helper()
		sigs, err := plan9asmSigsForPkg(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		return sigs
	}
	if _, ok := load(first)[path+".Foo"]; !ok {
		t.Fatal("first context did not translate Foo")
	}
	if len(load(second)) != 0 {
		t.Fatal("signature cache leaked between build contexts")
	}
	// Removing the source distinguishes a same-context cache hit from a retry.
	delete(first.buildConf.Overlay, asm)
	if _, ok := load(first)[path+".Foo"]; !ok {
		t.Fatal("same context did not reuse its cached signatures")
	}
}

func newPlan9AsmSignatureTestContext(t *testing.T) (*context, string, string) {
	t.Helper()
	const path = "example.com/asm"
	dir := t.TempDir()
	asm := filepath.Join(dir, "foo_amd64.s")
	typesPkg := types.NewPackage(path, "asm")
	typesPkg.Scope().Insert(types.NewFunc(token.NoPos, typesPkg, "Foo", types.NewSignatureType(nil, nil, nil, nil, nil, false)))
	pkg := &packages.Package{ID: path, PkgPath: path, Types: typesPkg, Fset: token.NewFileSet(), Dir: dir, OtherFiles: []string{asm}}
	return &context{
		plan9asmReady: true, plan9asmMode: plan9asmEnvAll,
		buildConf: &Config{Goos: "linux", Goarch: "amd64", Overlay: make(map[string][]byte)},
		pkgs:      map[*packages.Package]Package{pkg: {Package: pkg}},
	}, path, asm
}

func TestPlan9AsmSignatureCacheConcurrentLookup(t *testing.T) {
	ctx, path, asm := newPlan9AsmSignatureTestContext(t)
	ctx.buildConf.Overlay[asm] = []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\nRET\n")
	// Backend tasks own distinct contexts, so production does not concurrently
	// populate one entry. Exercise concurrent reads of the published map, which
	// is the operation protected by the cache and its read-only contract.
	if sigs, err := plan9asmSigsForPkg(ctx, path); err != nil || len(sigs) != 1 {
		t.Fatalf("populate signature cache: %v, %v", sigs, err)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 8 {
				sigs, err := plan9asmSigsForPkg(ctx, path)
				_, hasFoo := sigs[path+".Foo"]
				if err != nil || len(sigs) != 1 || !hasFoo {
					t.Errorf("concurrent signature lookup: %v, %v", sigs, err)
				}
			}
		})
	}
	wg.Wait()
}
