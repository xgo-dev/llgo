package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEqualitySourceOwnerSignatureBoundaries(t *testing.T) {
	const assembly = "TEXT runtime·memequal<ABIInternal>(SB),4,$0-25\nRET\n" +
		"TEXT runtime·memequal_varlen<ABIInternal>(SB),4,$0-17\nRET\n"
	for _, mutation := range []string{
		"aliased-unsafe", "unnamed-parameters", "generic", "no-result", "multiple-results",
		"wrong-result", "integer-pointer", "wrong-pointer-qualifier", "wrong-size", "missing-public-owner", "malformed-body",
	} {
		t.Run(mutation, func(t *testing.T) {
			body := runtimeEqualityOwnerPatchFixture
			want := "missing, duplicate or incompatible real equality Go body"
			switch mutation {
			case "aliased-unsafe":
				body = strings.ReplaceAll(strings.Replace(body, "import \"unsafe\"", "import raw \"unsafe\"", 1), "unsafe.Pointer", "raw.Pointer")
				want = ""
			case "unnamed-parameters":
				// Source-identity validation is not execution evidence. Both
				// declaration spellings must still describe the same ABI.
				body = strings.ReplaceAll(body, "bool { return a == b }", "bool { return true }")
				body = strings.Replace(body, "a, b unsafe.Pointer, size uintptr", "unsafe.Pointer, unsafe.Pointer, uintptr", 1)
				body = strings.Replace(body, "a, b unsafe.Pointer", "unsafe.Pointer, unsafe.Pointer", 1)
				want = ""
			case "generic":
				body = strings.Replace(body, "func abigen_runtime_memequal(", "func abigen_runtime_memequal[T any](", 1)
			case "no-result":
				body = strings.Replace(body, "bool { return a == b }", "{ }", 1)
			case "multiple-results":
				body = strings.Replace(body, "bool { return a == b }", "(bool, bool) { return true, false }", 1)
			case "wrong-result":
				body = strings.Replace(body, "bool { return a == b }", "uint32 { return 0 }", 1)
			case "integer-pointer":
				body = strings.Replace(body, "a, b unsafe.Pointer", "a, b uintptr", 1)
			case "wrong-pointer-qualifier":
				body = strings.Replace(body, "a, b unsafe.Pointer", "a, b other.Pointer", 1)
			case "wrong-size":
				body = strings.Replace(body, "size uintptr", "size uint32", 1)
			case "missing-public-owner":
				body = strings.Split(body, "//go:linkname abigen_runtime_memequal_varlen")[0]
				want = "both equality public entries require selected real Go bodies"
			case "malformed-body":
				body, want = "package", "expected"
			}
			err := validateEqualitySourcePatchOwners([]byte(assembly), [][]byte{[]byte(body)})
			if want == "" {
				if err != nil {
					t.Fatal("equivalent complete owner ABI rejected", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("owner signature %s: got %v, want %q", mutation, err, want)
			}
		})
	}
}

func TestEqualitySourceOwnerRequiresCurrentPhysicalSource(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		t.Run(map[bool]string{false: "initial", true: "generated-view"}[reuse], func(t *testing.T) {
			goroot, runtimeDir := t.TempDir(), t.TempDir()
			dir := filepath.Join(goroot, "src", "internal", "bytealg")
			patchDir := filepath.Join(runtimeDir, "_patch", "internal", "bytealg")
			for _, directory := range []string{dir, patchDir} {
				if err := os.MkdirAll(directory, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			assemblyLanguageTestWrite(t, patchDir, "equal_native.go", runtimeEqualityOwnerPatchFixture)
			assemblyLanguageTestWrite(t, dir, "other.go", "package bytealg\n")
			filename := filepath.Join(dir, "equal_amd64.s")
			ctx := sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.2"}
			var selected map[string][]byte
			want := "read runtime equality assembly owner"
			if reuse {
				assemblyLanguageTestWrite(t, dir, "equal_amd64.s",
					"TEXT runtime·memequal<ABIInternal>(SB),4,$0-25\nRET\n"+
						"TEXT runtime·memequal_varlen<ABIInternal>(SB),4,$0-17\nRET\n")
				changed, view, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, "internal/bytealg", ctx)
				if err != nil || !changed {
					t.Fatal("complete original owner prerequisite", err)
				}
				selected = view
				if err := os.Rename(filename, filename+".unused"); err != nil {
					t.Fatal(err)
				}
				want = "reread runtime equality assembly owner"
			}
			changed, view, files, err := applySourcePatchForPkg(selected, nil, runtimeDir, goroot, "internal/bytealg", ctx)
			if err == nil || !strings.Contains(err.Error(), want) || changed || view != nil || files != nil {
				t.Fatal("missing current assembly acquired a replacement view", changed, view, files, err)
			}
		})
	}
}
