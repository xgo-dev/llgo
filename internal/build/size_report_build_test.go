package build

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/sizereport"
	"golang.org/x/tools/go/packages"
)

// A valid empty function is enough to exercise build-output plumbing; binary
// parser fixtures and malformed-input coverage live in internal/sizereport.
func sizeWasmFixture() []byte {
	return []byte{0, 'a', 's', 'm', 1, 0, 0, 0, 1, 4, 1, 0x60, 0, 0, 3, 2, 1, 0, 10, 4, 1, 2, 0, 0x0b}
}

type sizeFailWriter struct{}

func (sizeFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestFinalSizeEmscriptenArtifacts(t *testing.T) {
	dir := t.TempDir()
	out := &OutFmtDetails{Out: filepath.Join(dir, "app.mjs"), PCLN: filepath.Join(dir, "app.pclntab")}
	raw := sizeWasmFixture()
	for path, data := range map[string][]byte{out.Out: []byte("javascript"), filepath.Join(dir, "app.wasm"): raw, out.PCLN: []byte("pcln")} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	conf := &Config{Mode: ModeBuild, SizeReport: true, SizeFormat: "json", Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactNone}
	var output bytes.Buffer
	if err := reportBuildOutputs(conf, out, nil, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Version   int
		Stage     string
		Binary    string
		FileSize  uint64 `json:"file_size"`
		Artifacts []Artifact
	}
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Version != 1 || payload.Stage != "final" || !strings.HasSuffix(payload.Binary, "app.wasm") || payload.FileSize != uint64(len(raw)) || len(payload.Artifacts) != 3 {
		t.Fatalf("final report: %s", output.String())
	}
	if err := os.Remove(out.PCLN); err != nil {
		t.Fatal(err)
	}
	if err := reportBuildOutputs(conf, out, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing final artifact must fail report")
	}
}

func TestSizeReportConfiguration(t *testing.T) {
	if err := ensureSizeReporting(&Config{SizeReport: true, SizeFormat: "invalid"}); err == nil {
		t.Fatal("invalid format accepted")
	}
	if err := ensureSizeReporting(&Config{SizeReport: true, SizeLevel: "invalid"}); err == nil {
		t.Fatal("invalid level accepted")
	}
	conf := &Config{SizeReport: true, SizeFormat: "JSON", SizeLevel: "FULL"}
	if err := ensureSizeReporting(conf); err != nil || conf.SizeFormat != "json" || conf.SizeLevel != "full" {
		t.Fatalf("normalization: %+v %v", conf, err)
	}
}

func TestSizeReportBuildErrors(t *testing.T) {
	dir := t.TempDir()
	conf := &Config{Mode: ModeRun, SizeReport: true}
	if err := reportBuildOutputs(conf, &OutFmtDetails{}, nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	conf.Mode = ModeBuild
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: filepath.Join(t.TempDir(), "missing")}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing binary accepted")
	}
	wasmPath := filepath.Join(dir, "future-version.wasm")
	if err := os.WriteFile(wasmPath, []byte{0, 'a', 's', 'm', 2, 0, 0, 0}, 0o600); err != nil {
		t.Fatal(err)
	}
	conf = &Config{Mode: ModeBuild, SizeReport: true, DebugArtifactMode: DebugArtifactNone}
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: wasmPath}, nil, io.Discard, io.Discard); err == nil || !strings.Contains(err.Error(), "size report: invalid WebAssembly header") {
		t.Fatalf("invalid binary report must fail the build: %v", err)
	}
}

func TestSizeReportPackages(t *testing.T) {
	pkgs := []Package{nil, &aPackage{}, &aPackage{Package: &packages.Package{
		PkgPath: "example.com/module/pkg", Module: &packages.Module{Path: "example.com/module"},
	}}, &aPackage{Package: &packages.Package{PkgPath: "runtime"}}}
	got := sizeReportPackages(pkgs)
	want := []sizereport.Package{{Path: "example.com/module/pkg", Module: "example.com/module"}, {Path: "runtime"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("package metadata = %+v, want %+v", got, want)
	}
}

func TestSizeReportNativeFallback(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Mach-O tool fallback is exercised on Darwin")
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	conf := &Config{Mode: ModeBuild, SizeReport: true, SizeFormat: "json", Goos: "darwin", DebugArtifactMode: DebugArtifactNone}
	var output bytes.Buffer
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Format string
		Total  struct{ Code uint64 }
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || report.Format != "native" || report.Total.Code == 0 {
		t.Fatalf("native fallback output = %s, error = %v", output.String(), err)
	}
	t.Setenv("LLVM_CONFIG", filepath.Join(t.TempDir(), "missing-llvm-config"))
	t.Setenv("PATH", t.TempDir())
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, io.Discard, io.Discard); err == nil || !strings.Contains(err.Error(), "llvm-readelf") {
		t.Fatalf("missing native tool must fail a requested report: %v", err)
	}
}
