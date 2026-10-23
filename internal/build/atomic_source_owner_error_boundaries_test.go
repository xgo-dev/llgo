package build

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func atomicSourceBoundaryInputs() (assembly, declarations, bodies string) {
	var asm, decl, body strings.Builder
	decl.WriteString("package atomic\nimport \"unsafe\"\n")
	body.WriteString("package atomic\nimport \"unsafe\"\n")
	for _, row := range runtimeAtomicRows() {
		fmt.Fprintf(&asm, "TEXT ·%s(SB),4,$0\nRET\n", row.name)
		fmt.Fprintln(&decl, row.declaration())
		fmt.Fprintf(&body, "%s { panic(\"source identity fixture, not execution evidence\") }\n", row.declaration())
	}
	return asm.String(), decl.String(), body.String()
}

func TestAtomicSourceOwnerParseBoundaries(t *testing.T) {
	for _, mutation := range []string{"declarations", "bodies", "duplicate-body", "extra-public-body", "semicolon-text"} {
		t.Run(mutation, func(t *testing.T) {
			assembly, declarations, bodies := atomicSourceBoundaryInputs()
			check := func() error {
				return validateAtomicSourcePatchOwners([]byte(assembly), [][]byte{[]byte(declarations)}, [][]byte{[]byte(bodies)}, "ARM64", 49)
			}
			if err := check(); err != nil {
				t.Fatal("complete independent source-identity fixture", err)
			}
			want := "expected"
			switch mutation {
			case "declarations":
				declarations = "package"
			case "bodies":
				bodies = "package"
			case "duplicate-body":
				bodies += "\nfunc Load8(ptr *uint8) uint8 { return 0 }\n"
				want = "duplicate ARM64 atomic Go owner"
			case "extra-public-body":
				bodies += "\nfunc Unexpected() {}\n"
				want = "unowned public ARM64 atomic replacement"
			case "semicolon-text":
				assembly = strings.Replace(assembly, ",$0\n", ",$0; TEXT ·Another(SB),4,$0\n", 1)
				want = "unrecognized ARM64 atomic TEXT owner"
			}
			if err := check(); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("changed atomic owner %s: got %v, want %q", mutation, err, want)
			}
		})
	}
}

func TestAtomicSourceOwnerCurrentFileBoundaries(t *testing.T) {
	for _, mutation := range []string{"missing-assembly", "missing-declarations", "missing-marker-owner", "reread-assembly", "reread-declarations", "changed-generated-declarations"} {
		t.Run(mutation, func(t *testing.T) {
			assembly, declarations, bodies := atomicSourceBoundaryInputs()
			goroot, runtimeDir := t.TempDir(), t.TempDir()
			dir := filepath.Join(goroot, "src", "internal", "runtime", "atomic")
			patchDir := filepath.Join(runtimeDir, "_patch", "internal", "runtime", "atomic")
			for _, directory := range []string{dir, patchDir} {
				if err := os.MkdirAll(directory, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			asm := assemblyLanguageTestWrite(t, dir, "atomic_arm64.s", assembly)
			stubs := assemblyLanguageTestWrite(t, dir, "stubs.go", declarations)
			assemblyLanguageTestWrite(t, dir, "atomic_arm64.go", "package atomic\n")
			assemblyLanguageTestWrite(t, patchDir, "atomic_arm64.go", bodies)
			ctx := sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.27.2"}
			changed, original, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, "internal/runtime/atomic", ctx)
			if err != nil || !changed {
				t.Fatal("complete original source owner prerequisite", err)
			}
			var selected map[string][]byte
			want := ""
			missing := ""
			switch mutation {
			case "missing-assembly":
				missing = asm
			case "missing-declarations":
				missing = stubs
			case "missing-marker-owner":
				selected = cloneOverlay(original)
				delete(selected, filepath.Join(dir, "z_llgo_patch_atomic_arm64.go"))
				want = "atomic marker lacks its exact generated owner"
			case "reread-assembly":
				selected, missing = original, asm
			case "reread-declarations":
				selected, missing = original, stubs
			case "changed-generated-declarations":
				selected = cloneOverlay(original)
				selected[stubs] = append(bytes.Clone(selected[stubs]), []byte("\nvar extraSelected int\n")...)
				want = "atomic generated declaration view changed"
			}
			if missing != "" {
				if err := os.Rename(missing, missing+".unused"); err != nil {
					t.Fatal(err)
				}
				want = filepath.Base(missing)
			}
			changed, view, files, err := applySourcePatchForPkg(selected, nil, runtimeDir, goroot, "internal/runtime/atomic", ctx)
			if err == nil || !strings.Contains(err.Error(), want) || changed || view != nil || files != nil {
				t.Fatal("incomplete current atomic source acquired ownership", changed, view, files, err)
			}
			if missing != "" && !errors.Is(err, os.ErrNotExist) {
				t.Fatal("missing source error identity was lost", err)
			}
		})
	}
}
