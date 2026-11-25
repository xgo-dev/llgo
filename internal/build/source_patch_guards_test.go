package build

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha256Hex(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func TestAtomicSourcePatchValidationGuards(t *testing.T) {
	validAsm := []byte("TEXT ·Load(SB), NOSPLIT, $0-8\n\tRET\nTEXT ·Store(SB), NOSPLIT, $0-8\n\tRET\n")
	validDecl := []byte("package atomic\nfunc Load(p *uint32) uint32\nfunc Store(p *uint32, v uint32)\n")
	validBody := []byte("package atomic\nfunc Load(p *uint32) uint32 { return *p }\nfunc Store(p *uint32, v uint32) { *p = v }\n")

	if err := validateAtomicSourcePatchOwners(validAsm, [][]byte{validDecl}, [][]byte{validBody}, "ARM64", 2); err != nil {
		t.Fatalf("valid atomic owners rejected: %v", err)
	}

	cases := []struct {
		name  string
		asm   string
		decls [][]byte
		body  [][]byte
		want  int
	}{
		{
			name:  "multiple TEXT on one line",
			asm:   "TEXT ·Load(SB), NOSPLIT, $0-0; TEXT ·Store(SB), NOSPLIT, $0-0\n",
			decls: [][]byte{validDecl},
			body:  [][]byte{validBody},
			want:  2,
		},
		{
			name:  "unrecognized TEXT format",
			asm:   "TEXT foo(SB), NOSPLIT, $0-0\n",
			decls: [][]byte{validDecl},
			body:  [][]byte{validBody},
			want:  1,
		},
		{
			name:  "private TEXT owner",
			asm:   "TEXT ·load(SB), NOSPLIT, $0-0\n",
			decls: [][]byte{validDecl},
			body:  [][]byte{validBody},
			want:  1,
		},
		{
			name:  "duplicate TEXT owner",
			asm:   "TEXT ·Load(SB), NOSPLIT, $0-0\nTEXT ·Load(SB), NOSPLIT, $0-0\n",
			decls: [][]byte{validDecl},
			body:  [][]byte{validBody},
			want:  2,
		},
		{
			name:  "wrong entry count",
			asm:   string(validAsm),
			decls: [][]byte{validDecl},
			body:  [][]byte{validBody},
			want:  3,
		},
		{
			name:  "declaration syntax error",
			asm:   string(validAsm),
			decls: [][]byte{[]byte("package atomic\nfunc (")},
			body:  [][]byte{validBody},
			want:  2,
		},
		{
			name:  "duplicate declaration owner",
			asm:   string(validAsm),
			decls: [][]byte{validDecl, []byte("package atomic\nfunc Load(p *uint32) uint32\n")},
			body:  [][]byte{validBody},
			want:  2,
		},
		{
			name:  "body syntax error",
			asm:   string(validAsm),
			decls: [][]byte{validDecl},
			body:  [][]byte{[]byte("package atomic\nfunc (")},
			want:  2,
		},
		{
			name:  "empty replacement body",
			asm:   string(validAsm),
			decls: [][]byte{validDecl},
			body:  [][]byte{[]byte("package atomic\nfunc Load(p *uint32) uint32 {}\nfunc Store(p *uint32, v uint32) { *p = v }\n")},
			want:  2,
		},
		{
			name:  "signature mismatch",
			asm:   string(validAsm),
			decls: [][]byte{validDecl},
			body:  [][]byte{[]byte("package atomic\nfunc Load(p *uint64) uint64 { return *p }\nfunc Store(p *uint32, v uint32) { *p = v }\n")},
			want:  2,
		},
		{
			name:  "unowned public replacement",
			asm:   string(validAsm),
			decls: [][]byte{validDecl},
			body:  [][]byte{[]byte("package atomic\nfunc Load(p *uint32) uint32 { return *p }\nfunc Store(p *uint32, v uint32) { *p = v }\nfunc Extra() {}\n")},
			want:  2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAtomicSourcePatchOwners([]byte(tc.asm), tc.decls, tc.body, "ARM64", tc.want); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestEqualitySourcePatchValidationGuards(t *testing.T) {
	validAsm := []byte("TEXT runtime·memequal(SB), NOSPLIT, $0-25\n\tRET\nTEXT runtime·memequal_varlen(SB), NOSPLIT, $0-17\n\tRET\nTEXT memeqbody<>(SB), NOSPLIT, $0-0\n\tRET\n")
	validBody := []byte(`package bytealg
import u "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b u.Pointer, size uintptr) bool { return a == b && size == 0 }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b u.Pointer) bool { return a == b }
`)
	if err := validateEqualitySourcePatchOwners(validAsm, [][]byte{validBody}); err != nil {
		t.Fatalf("valid equality owners with aliased unsafe rejected: %v", err)
	}

	cases := []struct {
		name string
		asm  string
		body string
	}{
		{
			name: "double TEXT line",
			asm:  "TEXT runtime·memequal(SB), NOSPLIT, $0-0; TEXT runtime·memequal_varlen(SB), NOSPLIT, $0-0\n",
			body: string(validBody),
		},
		{
			name: "unknown public TEXT entry",
			asm:  string(validAsm) + "TEXT runtime·extra(SB), NOSPLIT, $0-0\n",
			body: string(validBody),
		},
		{
			name: "incomplete public assembly group",
			asm:  "TEXT runtime·memequal(SB), NOSPLIT, $0-25\n\tRET\n",
			body: string(validBody),
		},
		{
			name: "body parse error",
			asm:  string(validAsm),
			body: "package bytealg\nfunc (",
		},
		{
			name: "missing llgo:env on varlen",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b unsafe.Pointer, size uintptr) bool { return true }
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b unsafe.Pointer) bool { return true }
`,
		},
		{
			name: "missing one body",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b unsafe.Pointer, size uintptr) bool { return true }
`,
		},
		{
			name: "wrong return type",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b unsafe.Pointer, size uintptr) int { return 0 }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b unsafe.Pointer) bool { return true }
`,
		},
		{
			name: "missing results",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b unsafe.Pointer, size uintptr) { _ = size }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b unsafe.Pointer) bool { return true }
`,
		},
		{
			name: "non-selector pointer param",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a uintptr, b unsafe.Pointer, size uintptr) bool { return true }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b unsafe.Pointer) bool { return true }
`,
		},
		{
			name: "wrong selector qualifier",
			asm:  string(validAsm),
			body: `package bytealg
//go:linkname memequal runtime.memequal
func memequal(a, b other.Pointer, size uintptr) bool { return true }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b other.Pointer) bool { return true }
`,
		},
		{
			name: "wrong third param type",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(a, b unsafe.Pointer, size int) bool { return true }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(a, b unsafe.Pointer) bool { return true }
`,
		},
		{
			name: "unnamed param signature check",
			asm:  string(validAsm),
			body: `package bytealg
import "unsafe"
//go:linkname memequal runtime.memequal
func memequal(unsafe.Pointer, unsafe.Pointer, uintptr) bool { return true }
//llgo:env
//go:linkname memequalVarlen runtime.memequal_varlen
func memequalVarlen(unsafe.Pointer, unsafe.Pointer, uintptr) bool { return true }
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateEqualitySourcePatchOwners([]byte(tc.asm), [][]byte{[]byte(tc.body)}); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestWindowsStdCallSourcePatchViewGuards(t *testing.T) {
	const patchName = "syscall_windows_test.go"
	const marker = "// replaced by LLGo source patch\n"
	dir := t.TempDir()
	origGo := []byte("package windows\nfunc StdCall(fn uintptr) uintptr\nfunc AsmStdCallAddr() uintptr\nfunc asmstdcall(fn uintptr)\n")
	origAsm := []byte("TEXT ·asmstdcall(SB), NOSPLIT, $0-8\n\tRET\n")
	validBody := []byte(`package windows
func StdCall(fn uintptr) uintptr { return llgoStdCall(fn) }
func AsmStdCallAddr() uintptr { return llgoStdCallAddr() }
func llgoStdCall(fn uintptr) uintptr { return fn }
func llgoStdCallAddr() uintptr { return 1 }
`)
	goPath := filepath.Join(dir, "syscall_windows.go")
	asmPath := filepath.Join(dir, "asm_windows_amd64.s")
	if err := os.WriteFile(goPath, origGo, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asmPath, origAsm, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored_test.go"), []byte("package windows\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other_linux.go"), []byte("package windows\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	windowsStdCallReviewedSourcesByPatch[patchName] = map[string]string{
		"syscall_windows.go":  sha256Hex(origGo),
		"asm_windows_amd64.s": sha256Hex(origAsm),
	}
	defer delete(windowsStdCallReviewedSourcesByPatch, patchName)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := build.Default
	ctx.GOOS = "windows"
	ctx.GOARCH = "amd64"
	wantSkips := map[string]struct{}{
		"StdCall": {}, "AsmStdCallAddr": {}, "asmstdcall": {}, "llgoStdCall": {}, "llgoStdCallAddr": {},
	}
	readDisk := func(p string) ([]byte, error) { return os.ReadFile(p) }

	// Happy path (fresh).
	view, err := windowsStdCallSourcePatchView(dir, entries, ctx, readDisk, nil, patchName, validBody, wantSkips)
	if err != nil || string(view[asmPath]) != marker {
		t.Fatalf("fresh windowsStdCallSourcePatchView failed: view=%v err=%v", view, err)
	}

	// Happy path (reused overlay).
	filteredGo, _, err := filterSourcePatchFile(origGo, wantSkips)
	if err != nil {
		t.Fatal(err)
	}
	reusedOverlay := map[string][]byte{
		asmPath: []byte(marker),
		goPath:  filteredGo,
		filepath.Join(dir, "z_llgo_patch_"+patchName): validBody,
	}
	readOverlay := func(p string) ([]byte, error) {
		if b, ok := reusedOverlay[p]; ok {
			return b, nil
		}
		return os.ReadFile(p)
	}
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, readOverlay, reusedOverlay, patchName, validBody, wantSkips); err != nil {
		t.Fatalf("reused windowsStdCallSourcePatchView failed: %v", err)
	}

	// Guard errors:
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, readDisk, nil, "unknown.go", validBody, wantSkips); err == nil {
		t.Fatal("expected error for unknown patchName")
	}
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, func(string) ([]byte, error) { return nil, errors.New("boom") }, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error when reading asm fails")
	}
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, readOverlay, map[string][]byte{}, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error when reused marker lacks matching body")
	}
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, readDisk, nil, patchName, validBody, map[string]struct{}{"StdCall": {}}); err == nil {
		t.Fatal("expected error when skips changed")
	}
	// Changed generated Go view on reused overlay.
	badReused := map[string][]byte{
		asmPath: []byte(marker),
		goPath:  []byte("package windows\n// tampered\n"),
		filepath.Join(dir, "z_llgo_patch_"+patchName): validBody,
	}
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if b, ok := badReused[p]; ok {
			return b, nil
		}
		return os.ReadFile(p)
	}, badReused, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error when reused Go view was tampered")
	}
	// Hash mismatch.
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if p == goPath {
			return []byte("package windows\n// modified\n"), nil
		}
		return os.ReadFile(p)
	}, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error on unreviewed hash")
	}
	// Extra unreviewed assembly file.
	extraAsm := filepath.Join(dir, "extra_windows_amd64.s")
	if err := os.WriteFile(extraAsm, []byte("TEXT ·x(SB),$0\nRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entriesExtra, _ := os.ReadDir(dir)
	if _, err := windowsStdCallSourcePatchView(dir, entriesExtra, ctx, readDisk, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error on unreviewed extra .s file")
	}
	_ = os.Remove(extraAsm)

	// Remaining Go consumer of private asmstdcall.
	consumerPath := filepath.Join(dir, "consumer_windows.go")
	if err := os.WriteFile(consumerPath, []byte("package windows\nfunc use() { asmstdcall(0) }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	entriesConsumer, _ := os.ReadDir(dir)
	if _, err := windowsStdCallSourcePatchView(dir, entriesConsumer, ctx, readDisk, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error for remaining consumer of asmstdcall")
	}
	// Invalid Go syntax in unreviewed Go file.
	if err := os.WriteFile(consumerPath, []byte("package windows\nfunc ("), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := windowsStdCallSourcePatchView(dir, entriesConsumer, ctx, readDisk, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error for invalid Go file")
	}
	_ = os.Remove(consumerPath)

	// Incomplete group (seen != 2).
	if _, err := windowsStdCallSourcePatchView(dir, nil, ctx, readDisk, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error for incomplete source group")
	}
	// Read error inside directory loop.
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if p == goPath {
			return nil, errors.New("go read boom")
		}
		return os.ReadFile(p)
	}, nil, patchName, validBody, wantSkips); err == nil {
		t.Fatal("expected error when reading Go file inside loop fails")
	}
	// Reused overlay when original file on disk was deleted.
	if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if b, ok := reusedOverlay[p]; ok {
			return b, nil
		}
		return os.ReadFile(p)
	}, reusedOverlay, patchName, validBody, wantSkips); err != nil {
		t.Fatalf("sanity reused overlay failed: %v", err)
	}
	// Replacement body parse error / group count mismatch / missing body / signature mismatch.
	for _, badBody := range []string{
		"package windows\nfunc (",
		"package windows\nfunc StdCall(fn uintptr) uintptr { return fn }\n",
		"package windows\nfunc StdCall(fn uintptr) uintptr {}\nfunc AsmStdCallAddr() uintptr { return 1 }\nfunc llgoStdCall(fn uintptr) uintptr { return fn }\nfunc llgoStdCallAddr() uintptr { return 1 }\n",
		"package windows\nfunc StdCall(fn int) int { return fn }\nfunc AsmStdCallAddr() uintptr { return 1 }\nfunc llgoStdCall(fn uintptr) uintptr { return fn }\nfunc llgoStdCallAddr() uintptr { return 1 }\n",
	} {
		if _, err := windowsStdCallSourcePatchView(dir, entries, ctx, readDisk, nil, patchName, []byte(badBody), wantSkips); err == nil {
			t.Fatalf("expected error for bad replacement body:\n%s", badBody)
		}
	}
}

func TestGCScanSourcePatchViewGuards(t *testing.T) {
	const marker = "// replaced by LLGo source patch\n"
	dir := t.TempDir()
	files := map[string][]byte{
		"expand_amd64.s":  []byte("TEXT ·ExpandAVX512(SB), NOSPLIT, $0-8\n\tRET\n"),
		"scan_amd64.s":    []byte("TEXT ·scanSpanPackedAVX512(SB), NOSPLIT, $0-8\n\tRET\n"),
		"filter_amd64.s":  []byte("TEXT ·FilterNilAVX512(SB), NOSPLIT, $0-8\n\tRET\n"),
		"expand_amd64.go": []byte("package scan\nvar gcExpandersAVX512 [1]uintptr\nfunc ExpandAVX512(n int)\n"),
		"scan_amd64.go":   []byte("package scan\nfunc scanSpanPackedAVX512(n int)\n"),
		"filter_amd64.go": []byte("package scan\nfunc FilterNilAVX512(n int)\n"),
	}
	saved := gcScanReviewedSources
	synthetic := make(map[string][]string, len(files))
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
		synthetic[name] = []string{sha256Hex(content)}
	}
	gcScanReviewedSources = synthetic
	defer func() { gcScanReviewedSources = saved }()
	// Add ignored _test.go and non-matching OS file to exercise loop filters (lines 75, 82).
	_ = os.WriteFile(filepath.Join(dir, "ignored_test.go"), []byte("package scan\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "other_darwin.go"), []byte("package scan\n"), 0o600)

	validBody := []byte(`package scan
func llgoGCScanFactor(n int) int { return n }
func ExpandAVX512(n int) { _ = llgoGCScanFactor(n) }
func scanSpanPackedAVX512(n int) { _ = llgoGCScanFactor(n) }
func FilterNilAVX512(n int) { _ = llgoGCScanFactor(n) }
`)
	wantSkips := map[string]struct{}{
		"ExpandAVX512": {}, "scanSpanPackedAVX512": {}, "FilterNilAVX512": {},
		"llgoGCScanFactor": {}, "gcExpandersAVX512": {},
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := build.Default
	ctx.GOOS = "linux"
	ctx.GOARCH = "amd64"
	readDisk := func(p string) ([]byte, error) { return os.ReadFile(p) }

	if _, err := gcScanSourcePatchView(dir, entries, ctx, readDisk, nil, validBody, wantSkips); err != nil {
		t.Fatalf("valid gcScanSourcePatchView failed: %v", err)
	}
	// Happy reused overlay path (lines 93-103).
	validReused := map[string][]byte{
		filepath.Join(dir, "expand_amd64.s"):             []byte(marker),
		filepath.Join(dir, "scan_amd64.s"):               []byte(marker),
		filepath.Join(dir, "filter_amd64.s"):             []byte(marker),
		filepath.Join(dir, "z_llgo_patch_scan_amd64.go"): validBody,
	}
	for _, goName := range []string{"expand_amd64.go", "scan_amd64.go", "filter_amd64.go"} {
		filtered, _, ferr := filterSourcePatchFile(files[goName], wantSkips)
		if ferr != nil {
			t.Fatal(ferr)
		}
		validReused[filepath.Join(dir, goName)] = filtered
	}
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if b, ok := validReused[p]; ok {
			return b, nil
		}
		return os.ReadFile(p)
	}, validReused, validBody, wantSkips); err != nil {
		t.Fatalf("valid reused gcScanSourcePatchView failed: %v", err)
	}

	// Read error on assembly file.
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(string) ([]byte, error) { return nil, errors.New("io") }, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error on asm read failure")
	}
	// Partial reused markers.
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if filepath.Base(p) == "expand_amd64.s" {
			return []byte(marker), nil
		}
		return os.ReadFile(p)
	}, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for partial reused markers")
	}
	// Changed skips.
	if _, err := gcScanSourcePatchView(dir, entries, ctx, readDisk, nil, validBody, map[string]struct{}{}); err == nil {
		t.Fatal("expected error for changed skips")
	}
	// Tampered reused Go view.
	reused := map[string][]byte{
		filepath.Join(dir, "expand_amd64.s"):             []byte(marker),
		filepath.Join(dir, "scan_amd64.s"):               []byte(marker),
		filepath.Join(dir, "filter_amd64.s"):             []byte(marker),
		filepath.Join(dir, "z_llgo_patch_scan_amd64.go"): validBody,
		filepath.Join(dir, "expand_amd64.go"):            []byte("package scan\n// tampered\n"),
	}
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if b, ok := reused[p]; ok {
			return b, nil
		}
		return os.ReadFile(p)
	}, reused, validBody, wantSkips); err == nil {
		t.Fatal("expected error for tampered reused Go view")
	}
	// Extra unowned .s file.
	extraAsm := filepath.Join(dir, "new_amd64.s")
	_ = os.WriteFile(extraAsm, []byte("TEXT ·x(SB),$0\nRET\n"), 0o600)
	entriesExtra, _ := os.ReadDir(dir)
	if _, err := gcScanSourcePatchView(dir, entriesExtra, ctx, readDisk, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for unowned .s file")
	}
	_ = os.Remove(extraAsm)
	// Extra Go consumer of gcExpandersAVX512 + invalid syntax.
	consumer := filepath.Join(dir, "consumer_amd64.go")
	_ = os.WriteFile(consumer, []byte("package scan\nvar _ = gcExpandersAVX512\n"), 0o600)
	entriesCons, _ := os.ReadDir(dir)
	if _, err := gcScanSourcePatchView(dir, entriesCons, ctx, readDisk, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for remaining consumer of gcExpandersAVX512")
	}
	_ = os.WriteFile(consumer, []byte("package scan\nfunc ("), 0o600)
	if _, err := gcScanSourcePatchView(dir, entriesCons, ctx, readDisk, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for syntax error in Go file")
	}
	_ = os.Remove(consumer)
	// Incomplete source group.
	if _, err := gcScanSourcePatchView(dir, nil, ctx, readDisk, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for incomplete group")
	}
	// Unreviewed SHA-256 hash mismatch.
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if filepath.Base(p) == "expand_amd64.go" {
			return []byte("package scan\n// changed\n"), nil
		}
		return os.ReadFile(p)
	}, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error for unreviewed gcScan SHA-256 hash")
	}
	// Read error inside loop.
	if _, err := gcScanSourcePatchView(dir, entries, ctx, func(p string) ([]byte, error) {
		if filepath.Base(p) == "expand_amd64.go" {
			return nil, errors.New("read go boom")
		}
		return os.ReadFile(p)
	}, nil, validBody, wantSkips); err == nil {
		t.Fatal("expected error when reading Go file inside gcScan loop fails")
	}
	// Bad replacement bodies.
	for _, bad := range []string{
		"package scan\nfunc (",
		"package scan\nfunc ExpandAVX512(n int) { _ = n }\n",
		"package scan\nfunc llgoGCScanFactor(n int) int {}\nfunc ExpandAVX512(n int) { _ = n }\nfunc scanSpanPackedAVX512(n int) { _ = n }\nfunc FilterNilAVX512(n int) { _ = n }\n",
		"package scan\nfunc llgoGCScanFactor(n int) int { return n }\nfunc ExpandAVX512(n int) {}\nfunc scanSpanPackedAVX512(n int) { _ = n }\nfunc FilterNilAVX512(n int) { _ = n }\n",
		"package scan\nfunc llgoGCScanFactor(n int) int { return n }\nfunc ExpandAVX512(n uint64) { _ = n }\nfunc scanSpanPackedAVX512(n int) { _ = n }\nfunc FilterNilAVX512(n int) { _ = n }\n",
	} {
		if _, err := gcScanSourcePatchView(dir, entries, ctx, readDisk, nil, []byte(bad), wantSkips); err == nil {
			t.Fatalf("expected error for bad gcScan body:\n%s", bad)
		}
	}
}

func TestSourcePatchHelperEdgeCases(t *testing.T) {
	// parseSourcePatchBuildTags with separate "-tags" argument.
	if got := parseSourcePatchBuildTags([]string{"-tags", "alpha,beta", "-tags=beta,gamma"}); len(got) != 3 {
		t.Fatalf("parseSourcePatchBuildTags = %v, want 3 tags", got)
	}

	// releaseTagsForGoVersion error branches.
	for _, bad := range []string{"go1", "goX.2", "go1.Y", "go2.0", "go1.0"} {
		if _, err := releaseTagsForGoVersion(bad); err == nil {
			t.Fatalf("releaseTagsForGoVersion(%q) unexpectedly succeeded", bad)
		}
	}

	// packageStubSource missing package clause.
	if _, err := packageStubSource([]byte("// no package here\n")); err == nil {
		t.Fatal("expected packageStubSource error")
	}

	// collectSourcePatchDirectives / filterSourcePatchFile parse errors.
	if _, err := collectSourcePatchDirectives([]byte("package p\nfunc (")); err == nil {
		t.Fatal("expected collectSourcePatchDirectives error")
	}
	if _, _, err := filterSourcePatchFile([]byte("package p\nfunc ("), map[string]struct{}{"A": {}}); err == nil {
		t.Fatal("expected filterSourcePatchFile error")
	}

	// filterSourcePatchSpec mixed multi-name ValueSpec.
	filtered, changed, err := filterSourcePatchFile([]byte("package p\nvar A, B = 1, 2\n"), map[string]struct{}{"A": {}})
	if err != nil || !changed || !strings.Contains(string(filtered), "// var A, B = 1, 2") {
		t.Fatalf("mixed multi-name ValueSpec filter: changed=%v err=%v out=%q", changed, err, filtered)
	}

	// commentSourcePatchSpans edge cases (empty, clamping, overlapping, zero-length).
	src := []byte("alpha\nbeta\ngamma\n")
	if got := string(commentSourcePatchSpans(src, nil)); got != string(src) {
		t.Fatalf("empty spans mutated source: %q", got)
	}
	got := string(commentSourcePatchSpans(src, []sourcePatchSpan{
		{start: -5, end: 5},
		{start: 0, end: 8},
		{start: 0, end: 8},
		{start: 6, end: 6},
		{start: 10, end: 100},
	}))
	if !strings.Contains(got, "// alpha") || !strings.Contains(got, "// gamma") {
		t.Fatalf("unexpected commented output: %q", got)
	}

	// commentGroupsToSpans nil element skipped.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", "package p\n// doc\nfunc F() {}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if spans := commentGroupsToSpans(fset.File(file.Pos()), []*ast.CommentGroup{nil, file.Comments[0]}); len(spans) != 1 {
		t.Fatalf("commentGroupsToSpans len = %d, want 1", len(spans))
	}

	// recvPatchKey ParenExpr, SelectorExpr, and unsupported panic.
	if k := recvPatchKey(&ast.ParenExpr{X: &ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ast.NewIdent("T")}}); k != "T" {
		t.Fatalf("recvPatchKey = %q, want T", k)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic on unsupported receiver expression")
			}
		}()
		recvPatchKey(&ast.BasicLit{})
	}()

	// applySourcePatchForPkg guard errors with synthetic runtimeDir + goroot.
	root := t.TempDir()
	rtDir := filepath.Join(root, "rt")
	goRoot := filepath.Join(root, "goroot")
	// Missing stdlib dir returns (false, nil).
	_ = os.MkdirAll(filepath.Join(rtDir, "_patch", "internal", "missing"), 0o700)
	if changed, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/missing", sourcePatchBuildContext{goos: "linux", goarch: "amd64"}); err != nil || changed {
		t.Fatalf("missing stdlib dir: changed=%v err=%v", changed, err)
	}
	// Invalid goversion propagates from newSourcePatchMatchContext.
	_ = os.MkdirAll(filepath.Join(goRoot, "src", "internal", "missing"), 0o700)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/missing", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "bad"}); err == nil {
		t.Fatal("expected error for invalid goversion")
	}
	// internal/bytealg skipAll rejection & missing asm rejection & reused marker missing generated owner.
	bytealgPatch := filepath.Join(rtDir, "_patch", "internal", "bytealg")
	bytealgSrc := filepath.Join(goRoot, "src", "internal", "bytealg")
	_ = os.MkdirAll(bytealgPatch, 0o700)
	_ = os.MkdirAll(bytealgSrc, 0o700)
	_ = os.WriteFile(filepath.Join(bytealgPatch, "eq.go"), []byte("package bytealg\n//llgo:skipall\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/bytealg", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for bytealg skipall")
	}
	_ = os.WriteFile(filepath.Join(bytealgPatch, "eq.go"), []byte("package bytealg\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/bytealg", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for missing equal_amd64.s")
	}
	eqAsm := filepath.Join(bytealgSrc, "equal_amd64.s")
	_ = os.WriteFile(eqAsm, []byte("// replaced by LLGo source patch\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(map[string][]byte{}, nil, rtDir, goRoot, "internal/bytealg", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for equality marker without generated Go owner")
	}
	// Equality reused overlay with matching generated Go owner re-reads physical asm and rejects invalid content (source_patch.go:222-233).
	eqPatchInjected := buildInjectedSourcePatchFile(filepath.Join(bytealgPatch, "eq.go"), []byte("package bytealg\n"))
	eqBase := map[string][]byte{
		eqAsm: []byte("// replaced by LLGo source patch\n"),
		filepath.Join(bytealgSrc, "z_llgo_patch_eq.go"): eqPatchInjected,
	}
	if _, _, _, err := applySourcePatchForPkg(eqBase, nil, rtDir, goRoot, "internal/bytealg", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected validateEqualitySourcePatchOwners error on invalid physical equal_amd64.s")
	}
	_ = os.Remove(eqAsm)
	if _, _, _, err := applySourcePatchForPkg(eqBase, nil, rtDir, goRoot, "internal/bytealg", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected os.ReadFile error when physical equal_amd64.s was removed")
	}

	// internal/runtime/atomic skipAll, missing asm, reused marker mismatch, missing stubs.go,
	// tampered reused declaration view, and validation error (source_patch.go:250-292).
	atomicPatch := filepath.Join(rtDir, "_patch", "internal", "runtime", "atomic")
	atomicSrc := filepath.Join(goRoot, "src", "internal", "runtime", "atomic")
	_ = os.MkdirAll(atomicPatch, 0o700)
	_ = os.MkdirAll(atomicSrc, 0o700)
	_ = os.WriteFile(filepath.Join(atomicPatch, "a.go"), []byte("package atomic\n//llgo:skipall\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error for atomic skipall")
	}
	_ = os.WriteFile(filepath.Join(atomicPatch, "a.go"), []byte("package atomic\nfunc Load() {}\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error for missing atomic_arm64.s")
	}
	atomicAsm := filepath.Join(atomicSrc, "atomic_arm64.s")
	atomicInjected := buildInjectedSourcePatchFile(filepath.Join(atomicPatch, "a.go"), []byte("package atomic\nfunc Load() {}\n"))
	// Reused marker with wrong/missing generated body.
	if _, _, _, err := applySourcePatchForPkg(map[string][]byte{atomicAsm: []byte("// replaced by LLGo source patch\n")}, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error for reused atomic marker lacking generated owner")
	}
	// Reused marker with matching body, physical atomic_arm64.s missing on disk.
	atomicBase := map[string][]byte{
		atomicAsm: []byte("// replaced by LLGo source patch\n"),
		filepath.Join(atomicSrc, "z_llgo_patch_a.go"): atomicInjected,
	}
	if _, _, _, err := applySourcePatchForPkg(atomicBase, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error when physical atomic_arm64.s is missing")
	}
	// Physical atomic_arm64.s exists, stubs.go missing -> readOverlay error (275-276).
	_ = os.WriteFile(atomicAsm, []byte("TEXT ·Load(SB), NOSPLIT, $0-0\nRET\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(atomicBase, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error when stubs.go is missing")
	}
	// stubs.go in overlay only (missing on disk) -> reused os.ReadFile error (280-281).
	stubsPath := filepath.Join(atomicSrc, "stubs.go")
	atomicArmGo := filepath.Join(atomicSrc, "atomic_arm64.go")
	atomicBase[stubsPath] = []byte("package atomic\n// tampered\n")
	if _, _, _, err := applySourcePatchForPkg(atomicBase, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error when physical stubs.go is missing")
	}
	// Physical stubs.go exists but overlay view was tampered -> generated declaration view changed (284-285).
	_ = os.WriteFile(stubsPath, []byte("package atomic\nfunc Load()\n"), 0o600)
	_ = os.WriteFile(atomicArmGo, []byte("package atomic\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(atomicBase, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected error for tampered reused atomic declaration view")
	}
	// Fresh (non-reused) atomic files failing validateAtomicSourcePatchOwners (291-292).
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/atomic", sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected validateAtomicSourcePatchOwners error on incomplete synthetic atomic_arm64.s")
	}
	// buildSourcePatchOverlayForGOROOT propagates applySourcePatchForPkg error (source_patch.go:50-51).
	if _, _, err := buildSourcePatchOverlayForGOROOT(nil, rtDir, goRoot, sourcePatchBuildContext{goos: "linux", goarch: "arm64", goversion: "go1.23.0"}); err == nil {
		t.Fatal("expected buildSourcePatchOverlayForGOROOT to propagate error")
	}

	// internal/runtime/syscall/windows & internal/runtime/gc/scan guard rejections.
	winPatch := filepath.Join(rtDir, "_patch", "internal", "runtime", "syscall", "windows")
	winSrc := filepath.Join(goRoot, "src", "internal", "runtime", "syscall", "windows")
	_ = os.MkdirAll(winPatch, 0o700)
	_ = os.MkdirAll(winSrc, 0o700)
	_ = os.WriteFile(filepath.Join(winPatch, "unknown.go"), []byte("package windows\n//llgo:skipall\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/syscall/windows", sourcePatchBuildContext{goos: "windows", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for windows syscall skipall")
	}
	_ = os.WriteFile(filepath.Join(winPatch, "unknown.go"), []byte("package windows\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/syscall/windows", sourcePatchBuildContext{goos: "windows", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for unknown windows syscall patch file")
	}

	scanPatch := filepath.Join(rtDir, "_patch", "internal", "runtime", "gc", "scan")
	scanSrc := filepath.Join(goRoot, "src", "internal", "runtime", "gc", "scan")
	_ = os.MkdirAll(scanPatch, 0o700)
	_ = os.MkdirAll(scanSrc, 0o700)
	_ = os.WriteFile(filepath.Join(scanPatch, "other.go"), []byte("package scan\n//llgo:skipall\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/gc/scan", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for gc/scan skipall")
	}
	_ = os.WriteFile(filepath.Join(scanPatch, "other.go"), []byte("package scan\n"), 0o600)
	if _, _, _, err := applySourcePatchForPkg(nil, nil, rtDir, goRoot, "internal/runtime/gc/scan", sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: "go1.27.0"}); err == nil {
		t.Fatal("expected error for gc/scan wrong filename")
	}
}
