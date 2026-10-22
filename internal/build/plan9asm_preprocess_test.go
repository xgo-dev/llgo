package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func plan9AsmTestPackage(ctx *context) *packages.Package {
	for pkg := range ctx.pkgs {
		return pkg
	}
	panic("missing test package")
}

func TestPlan9AsmHeaderChangesFingerprint(t *testing.T) {
	for _, kind := range []string{"package", "selected SDK", "overlay"} {
		t.Run(kind, func(t *testing.T) {
			ctx, _, asm := newPlan9AsmSignatureTestContext(t)
			ctx.mode = ModeBuild
			pkg := plan9AsmTestPackage(ctx)
			headerDir := pkg.Dir
			if kind == "selected SDK" {
				goroot := t.TempDir()
				if err := os.WriteFile(filepath.Join(goroot, "VERSION"), []byte("go1.27.2\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				headerDir = filepath.Join(goroot, "pkg", "include")
				if err := os.MkdirAll(headerDir, 0o700); err != nil {
					t.Fatal(err)
				}
				ctx.commands.environ = []string{"GOROOT=" + goroot}
			}
			header := filepath.Join(headerDir, "nested.h")
			wrapper := filepath.Join(pkg.Dir, "wrapper.h")
			ctx.buildConf.Overlay[asm] = []byte("#include \"wrapper.h\"\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
			ctx.buildConf.Overlay[wrapper] = []byte("#include \"nested.h\"\n")
			if err := os.WriteFile(header, []byte("#define VALUE 1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(header)
			if err != nil {
				t.Fatal(err)
			}
			fingerprint := func() string {
				t.Helper()
				m := newManifestBuilder()
				if err := ctx.collectPackageInputs(m, &aPackage{Package: pkg}); err != nil {
					t.Fatal(err)
				}
				return m.Fingerprint()
			}
			before := fingerprint()
			changed := []byte("#define VALUE 2\n")
			if kind == "overlay" {
				ctx.buildConf.Overlay[header] = changed
			} else {
				if err := os.WriteFile(header, changed, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(header, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
			}
			if before == fingerprint() {
				t.Fatal("assembly header content changed without invalidating the package cache")
			}
		})
	}
}

func TestPlan9AsmFingerprintSelectsActualPackage(t *testing.T) {
	ctx, _, asm := newPlan9AsmSignatureTestContext(t)
	ctx.mode = ModeBuild
	pkg := plan9AsmTestPackage(ctx)
	// A replacing alt package must not read the original's unused include.
	ctx.buildConf.Overlay[asm] = []byte("#include \"missing-original.h\"\n")
	altDir := t.TempDir()
	altAsm := filepath.Join(altDir, "alt.s")
	altHeader := filepath.Join(altDir, "alt.h")
	alt := &packages.Package{ID: "alt", PkgPath: "example.com/alt", Dir: altDir, Types: pkg.Types, OtherFiles: []string{altAsm}}
	ctx.buildConf.Overlay[altAsm] = []byte("#include \"alt.h\"\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
	ctx.buildConf.Overlay[altHeader] = []byte("#define VALUE 1\n")
	apkg := &aPackage{Package: pkg, AltPkg: &packages.Cached{Package: alt}}
	fingerprint := func() *manifestBuilder {
		t.Helper()
		m := newManifestBuilder()
		if err := ctx.collectPackageInputs(m, apkg); err != nil {
			t.Fatal(err)
		}
		return m
	}
	before := fingerprint()
	if len(before.pkg.Plan9AsmFiles) != 1 || before.pkg.Plan9AsmFiles[0].Path != altAsm {
		t.Fatalf("wrong assembly inputs: %+v", before.pkg.Plan9AsmFiles)
	}
	ctx.buildConf.Overlay[altHeader] = []byte("#define VALUE 2\n")
	if before.Fingerprint() == fingerprint().Fingerprint() {
		t.Fatal("alternate assembly header did not invalidate the cache")
	}
	ctx.mode = ModeGen
	delete(ctx.buildConf.Overlay, altHeader)
	if got := fingerprint().pkg.Plan9AsmFiles; len(got) != 0 {
		t.Fatalf("ModeGen fingerprinted assembly it does not compile: %+v", got)
	}
}

func TestPlan9AsmSignatureUsesIncludedSource(t *testing.T) {
	ctx, path, asm := newPlan9AsmSignatureTestContext(t)
	header := filepath.Join(filepath.Dir(asm), "implementation.h")
	ctx.buildConf.Overlay[asm] = []byte("#include \"implementation.h\"\n")
	ctx.buildConf.Overlay[header] = []byte("TEXT ·Foo(SB),4,$0-0\nRET\n")
	sigs, err := plan9asmSigsForPkg(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sigs[path+".Foo"]; !ok {
		t.Fatalf("included function missing from signature scan: %v", sigs)
	}
}

func TestPlan9AsmDisabledPolicyChecksIncludedText(t *testing.T) {
	ctx, _, asm := newPlan9AsmSignatureTestContext(t)
	ctx.plan9asmMode = plan9asmEnvNone
	ctx.buildConf.Overlay[asm] = []byte("#include \"implementation.h\"\n")
	ctx.buildConf.Overlay[filepath.Join(filepath.Dir(asm), "implementation.h")] = []byte("TEXT ·Foo(SB),4,$0-0\nRET\n")
	_, err := compilePkgSFiles(ctx, nil, plan9AsmTestPackage(ctx), false)
	if err == nil || !strings.Contains(err.Error(), "require plan9asm translation") {
		t.Fatalf("disabled translation silently dropped included TEXT: %v", err)
	}
}
