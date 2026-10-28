package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPlan9AsmSelectedSourceFailurePropagation(t *testing.T) {
	for _, consumer := range []string{"digest", "alternate-digest", "manifest", "compile", "signature"} {
		t.Run(consumer, func(t *testing.T) {
			ctx, path, assembly := newPlan9AsmSignatureTestContext(t)
			ctx.mode = ModeBuild
			pkg := plan9AsmTestPackage(ctx)
			ctx.buildConf.Overlay[assembly] = []byte("#include \"missing-selected-header.h\"\n")
			var err error
			switch consumer {
			case "digest":
				var digests []plan9AsmDigest
				digests, err = digestPackagePlan9Asm(ctx, &aPackage{Package: pkg}, []string{assembly})
				if digests != nil {
					t.Fatal("failed preprocessing supplied a cache digest", digests)
				}
			case "alternate-digest":
				alt := *pkg
				alt.ID, alt.PkgPath = "alternate", "example.com/alternate"
				var digests []plan9AsmDigest
				digests, err = digestPackagePlan9Asm(ctx, &aPackage{Package: pkg, AltPkg: &packages.Cached{Package: &alt}}, nil)
				if digests != nil {
					t.Fatal("failed selected alternate supplied a cache digest", digests)
				}
			case "manifest":
				manifest := newManifestBuilder()
				err = ctx.collectPackageInputs(manifest, &aPackage{Package: pkg})
				if len(manifest.pkg.Plan9AsmFiles) != 0 {
					t.Fatal("incomplete preprocessing entered the package cache key")
				}
			case "compile":
				objects, compileErr := compilePkgSFiles(ctx, nil, pkg, false)
				err = compileErr
				if len(objects) != 0 {
					t.Fatal("missing selected include produced an object", objects)
				}
			case "signature":
				signatures, signatureErr := plan9asmSigsForPkg(ctx, path)
				err = signatureErr
				if len(signatures) != 0 {
					t.Fatal("missing selected include produced signatures", signatures)
				}
			}
			if err == nil || !strings.Contains(err.Error(), "missing-selected-header.h") || !strings.Contains(err.Error(), "active include") {
				t.Fatal("selected source failure was hidden or replaced by a different failure", err)
			}
		})
	}
}

func TestCgoAssemblySelectedSourceErrors(t *testing.T) {
	for _, mutation := range []string{"missing-go", "malformed-go", "malformed-directive", "runtime-plan9-only", "alternate-source-error"} {
		t.Run(mutation, func(t *testing.T) {
			path := "example.com/source-errors"
			if mutation == "runtime-plan9-only" {
				path = "runtime/cgo"
			}
			ctx, pkg := assemblyLanguageTestFixture(t, path, true, false)
			assembly := assemblyLanguageTestWrite(t, pkg.Dir, "asm_amd64.s", "TEXT ·Foo(SB),4,$0-0\nRET\n")
			pkg.OtherFiles = []string{assembly}
			want := ""
			switch mutation {
			case "missing-go":
				if err := os.Rename(pkg.GoFiles[0], pkg.GoFiles[0]+".unused"); err != nil {
					t.Fatal(err)
				}
				want = "read selected Go source"
			case "malformed-go", "alternate-source-error":
				ctx.buildConf.Overlay[pkg.GoFiles[0]] = []byte("package fixture\nimport (")
				want = "inspect selected Go imports"
			case "malformed-directive":
				ctx.buildConf.Overlay[pkg.GoFiles[0]] = []byte("package fixture\n/*\n#cgo CFLAGS -I/missing\n*/\nimport \"C\"\n")
				want = "invalid cgo format"
			}
			if mutation == "alternate-source-error" {
				original := &packages.Package{ID: "original", PkgPath: "example.com/original", Dir: t.TempDir()}
				digests, err := digestPackagePlan9Asm(ctx, &aPackage{Package: original, AltPkg: &packages.Cached{Package: pkg}}, nil)
				if err == nil || !strings.Contains(err.Error(), want) || digests != nil {
					t.Fatal("alternate classification failure entered a digest", digests, err)
				}
				return
			}
			inputs, err := cgoAssemblyInputs(ctx, pkg, nil)
			if want == "" {
				if err != nil || len(inputs) != 0 {
					t.Fatal("runtime/cgo Plan 9 source was assigned a native compiler", inputs, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), want) || inputs != nil {
				t.Fatal("invalid selected Cgo source acquired a native compiler owner", inputs, err)
			}
		})
	}
}

func TestPlan9AsmDisabledAlternateDoesNotReadDiscardedSource(t *testing.T) {
	ctx, _, assembly := newPlan9AsmSignatureTestContext(t)
	ctx.plan9asmMode = plan9asmEnvNone
	pkg := plan9AsmTestPackage(ctx)
	pkg.PkgPath = "runtime"
	// The explicit runtime replacement owns execution, not this nonexistent
	// original. This is input selection, never a translated-assembly pass.
	digests, err := digestPlan9AsmFiles(ctx, pkg, []string{filepath.Join(filepath.Dir(assembly), "discarded.s")})
	if err != nil || digests != nil {
		t.Fatal("disabled original assembly was read despite its runtime owner", digests, err)
	}
}
