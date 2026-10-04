//go:build !llgo

package build

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestCollectArtifacts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	main := write("app.wasm", "main")
	glue := write("app.mjs", "javascript")
	htmlGlue := write("app.js", "html-glue")
	html := write("app.html", "html")
	host := write(wasmFSScriptName, "filesystem-host")
	elf := write("app.elf", "elf-data")
	dwarf := write("app.debug.wasm", "debug")
	pcln := write("app.wasm.pclntab", "pcln")
	bin := write("app.bin", "bin")
	hex := write("app.hex", "hex-data")

	tests := []struct {
		name string
		conf Config
		out  OutFmtDetails
		want []Artifact
	}{
		{
			name: "embedded",
			conf: Config{Goarch: "wasm", DebugArtifactMode: DebugArtifactEmbedded},
			out:  OutFmtDetails{Out: main},
			want: []Artifact{{Role: ArtifactRoleDebugDeployment, Format: "wasm", Path: main, Size: 4}},
		},
		{
			name: "external with runtime symbols",
			conf: Config{Goarch: "wasm", DebugArtifactMode: DebugArtifactExternal},
			out:  OutFmtDetails{Out: main, DWARF: dwarf, PCLN: pcln},
			want: []Artifact{
				{Role: ArtifactRoleDeployment, Format: "wasm", Path: main, Size: 4},
				{Role: ArtifactRoleDebug, Format: "wasm-dwarf", Path: dwarf, Size: 5},
				{Role: ArtifactRoleRuntimeSymbols, Format: "pclntab", Path: pcln, Size: 4},
			},
		},
		{
			name: "external Emscripten glue and module",
			conf: Config{Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactExternal},
			out:  OutFmtDetails{Out: glue, DWARF: dwarf},
			want: []Artifact{
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: glue, Size: 10},
				{Role: ArtifactRoleDeployment, Format: "wasm", Path: main, Size: 4},
				{Role: ArtifactRoleDebug, Format: "wasm-dwarf", Path: dwarf, Size: 5},
			},
		},
		{
			name: "Emscripten explicit Wasm includes sibling glue",
			conf: Config{Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactEmbedded},
			out:  OutFmtDetails{Out: main},
			want: []Artifact{
				{Role: ArtifactRoleDebugDeployment, Format: "wasm", Path: main, Size: 4},
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: glue, Size: 10},
			},
		},
		{
			name: "Emscripten HTML and sibling module",
			conf: Config{Target: "wasm", BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactEmbedded},
			out:  OutFmtDetails{Out: html},
			want: []Artifact{
				{Role: ArtifactRoleDeployment, Format: "html", Path: html, Size: 4},
				{Role: ArtifactRoleDebugDeployment, Format: "wasm", Path: main, Size: 4},
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: htmlGlue, Size: 9},
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: host, Size: 15},
			},
		},
		{
			name: "named Emscripten explicit Wasm owns browser host",
			conf: Config{Target: "emscripten-memory64", BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactEmbedded},
			out:  OutFmtDetails{Out: main},
			want: []Artifact{
				{Role: ArtifactRoleDebugDeployment, Format: "wasm", Path: main, Size: 4},
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: glue, Size: 10},
				{Role: ArtifactRoleDeployment, Format: "javascript", Path: host, Size: 15},
			},
		},
		{
			name: "host and deployment formats",
			conf: Config{Target: "cortex-m-qemu", DebugArtifactMode: DebugArtifactHost},
			out:  OutFmtDetails{Out: elf, Bin: bin, Hex: hex},
			want: []Artifact{
				{Role: ArtifactRoleDebug, Format: "elf", Path: elf, Size: 8},
				{Role: ArtifactRoleDeployment, Format: "bin", Path: bin, Size: 3},
				{Role: ArtifactRoleDeployment, Format: "hex", Path: hex, Size: 8},
			},
		},
		{
			name: "none",
			conf: Config{DebugArtifactMode: DebugArtifactNone},
			out:  OutFmtDetails{Out: main},
			want: []Artifact{{Role: ArtifactRoleDeployment, Format: "wasm", Path: main, Size: 4}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CollectArtifacts(&tt.conf, &tt.out)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CollectArtifacts() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCollectEmscriptenArtifactsRejectsIncompleteOutputs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		output  string
		target  string
		files   []string
		dirs    []string
		wantErr string
	}{
		{name: "missing primary glue", output: "app.mjs", files: []string{"app.wasm"}, wantErr: "stat deployment artifact"},
		{name: "missing HTML module", output: "app.html", files: []string{"app.html"}, wantErr: "stat debug+deployment artifact"},
		{name: "missing HTML glue", output: "app.html", files: []string{"app.html", "app.wasm"}, wantErr: "app.js"},
		{name: "missing named-target browser host", output: "app.mjs", target: "emscripten", files: []string{"app.mjs", "app.wasm"}, wantErr: wasmFSScriptName},
		{name: "optional glue is a directory", output: "app.wasm", files: []string{"app.wasm"}, dirs: []string{"app.mjs"}, wantErr: "not a regular file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range tc.dirs {
				if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			conf := &Config{Target: tc.target, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", DebugArtifactMode: DebugArtifactEmbedded}
			got, err := CollectArtifacts(conf, &OutFmtDetails{Out: filepath.Join(dir, tc.output)})
			if got != nil || err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("incomplete output produced a report: %v, %v", got, err)
			}
		})
	}
}

func TestCollectArtifactsValidation(t *testing.T) {
	if got, err := CollectArtifacts(nil, nil); err != nil || got != nil {
		t.Fatalf("CollectArtifacts(nil) = %#v, %v", got, err)
	}
	if _, err := CollectArtifacts(&Config{}, &OutFmtDetails{}); err == nil || !strings.Contains(err.Error(), "unresolved") {
		t.Fatalf("unresolved mode error = %v", err)
	}
	if _, err := CollectArtifacts(&Config{DebugArtifactMode: DebugArtifactNone}, &OutFmtDetails{}); err == nil || !strings.Contains(err.Error(), "path is empty") {
		t.Fatalf("empty primary path error = %v", err)
	}
	if _, err := CollectArtifacts(
		&Config{DebugArtifactMode: DebugArtifactNone},
		&OutFmtDetails{Out: filepath.Join(t.TempDir(), "missing")},
	); err == nil || !strings.Contains(err.Error(), "stat deployment artifact") {
		t.Fatalf("missing artifact error = %v", err)
	}
	dir := t.TempDir()
	if _, err := CollectArtifacts(
		&Config{DebugArtifactMode: DebugArtifactNone},
		&OutFmtDetails{Out: dir},
	); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory artifact error = %v", err)
	}

	main := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(main, []byte("main"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		mode DebugArtifactMode
		out  OutFmtDetails
		role ArtifactRole
	}{
		{name: "missing DWARF", mode: DebugArtifactExternal, out: OutFmtDetails{Out: main, DWARF: main + ".debug.wasm"}, role: ArtifactRoleDebug},
		{name: "missing runtime symbols", mode: DebugArtifactNone, out: OutFmtDetails{Out: main, PCLN: main + ".pclntab"}, role: ArtifactRoleRuntimeSymbols},
		{name: "missing deployment format", mode: DebugArtifactNone, out: OutFmtDetails{Out: main, Bin: main + ".bin"}, role: ArtifactRoleDeployment},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CollectArtifacts(&Config{DebugArtifactMode: tt.mode}, &tt.out)
			if err == nil || !strings.Contains(err.Error(), "stat "+string(tt.role)+" artifact") {
				t.Fatalf("CollectArtifacts() error = %v", err)
			}
		})
	}
}

func TestPrimaryArtifactFormat(t *testing.T) {
	tests := []struct {
		name string
		conf Config
		path string
		want string
	}{
		{name: "wasm architecture", conf: Config{Goarch: "wasm"}, path: "app", want: "wasm"},
		{name: "wasm extension", path: "app.WASM", want: "wasm"},
		{name: "target ELF", conf: Config{Target: "cortex-m-qemu"}, path: "app.elf", want: "elf"},
		{name: "archive", conf: Config{Goarch: "wasm", BuildMode: BuildModeCArchive}, path: "libapp.a", want: "archive"},
		{name: "Mach-O", conf: Config{Goos: "darwin", BuildMode: BuildModeCShared}, path: "libapp.dylib", want: "macho"},
		{name: "PE", conf: Config{Goos: "windows"}, path: "app.exe", want: "pe"},
		{name: "ELF", conf: Config{Goos: "linux", BuildMode: BuildModeCShared}, path: "libapp.so", want: "elf"},
		{name: "executable", conf: Config{BuildMode: BuildModeExe}, path: "app", want: "executable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := primaryArtifactFormat(&tt.conf, tt.path); got != tt.want {
				t.Fatalf("primaryArtifactFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReportBuildArtifacts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app with space")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	conf := &Config{DebugArtifactMode: DebugArtifactNone, DebugArtifactModeSet: true}
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, io.Discard, &report); err != nil {
		t.Fatal(err)
	}
	want := "llgo: artifact role=deployment format=executable size=4 path=" + strconv.Quote(path) + "\n"
	if got := report.String(); got != want {
		t.Fatalf("artifact report = %q, want %q", got, want)
	}

	report.Reset()
	conf.DebugArtifactModeSet = false
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, io.Discard, &report); err != nil || report.Len() != 0 {
		t.Fatalf("implicit artifact report = %q, %v", report.String(), err)
	}
	conf.DebugArtifactModeSet = true
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path + ".missing"}, nil, io.Discard, &report); err == nil {
		t.Fatal("reportBuildArtifacts() succeeded with a missing artifact")
	}

	conf.Target = "cortex-m-qemu"
	conf.DebugArtifactMode = DebugArtifactHost
	if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, io.Discard, &report); err != nil {
		t.Fatal(err)
	}
	if got := report.String(); !strings.Contains(got, "role=debug format=elf size=4") {
		t.Fatalf("target artifact report = %q", got)
	}
}

type artifactRemovingWriter struct {
	bytes.Buffer
	path string
}

func (w *artifactRemovingWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if err == nil && w.path != "" {
		err = os.Remove(w.path)
		w.path = ""
	}
	return n, err
}

func TestReportBuildOutputsSharesArtifacts(t *testing.T) {
	dir := t.TempDir()
	out := &OutFmtDetails{Out: filepath.Join(dir, "app.wasm"), PCLN: filepath.Join(dir, "app.pclntab")}
	if err := os.WriteFile(out.Out, sizeWasmFixture(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out.PCLN, []byte("symbols"), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := &Config{Mode: ModeBuild, SizeReport: true, SizeFormat: "json", Goarch: "wasm", DebugArtifactMode: DebugArtifactNone, DebugArtifactModeSet: true}
	// Removing a sidecar after the first report is written proves that the
	// second report consumes the same snapshot rather than re-statting files.
	output := artifactRemovingWriter{path: out.PCLN}
	var listing bytes.Buffer
	if err := reportBuildOutputs(conf, out, nil, &output, &listing); err != nil {
		t.Fatal(err)
	}
	var payload struct{ Artifacts []Artifact }
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Artifacts) != 2 || payload.Artifacts[1].Size != 7 {
		t.Fatalf("JSON artifact snapshot = %+v", payload.Artifacts)
	}
	if want := "role=runtime-symbols format=pclntab size=7 path=" + strconv.Quote(out.PCLN); !strings.Contains(listing.String(), want) {
		t.Fatalf("listing does not use the same snapshot: %s", listing.String())
	}
	if err := reportBuildOutputs(conf, out, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("a new build report must reject the missing sidecar")
	}
}

func TestReportBuildOutputsErrors(t *testing.T) {
	if err := reportBuildOutputs(nil, nil, nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := reportBuildOutputs(&Config{}, nil, nil, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app.wasm")
	if err := os.WriteFile(path, sizeWasmFixture(), 0o600); err != nil {
		t.Fatal(err)
	}
	conf := &Config{Mode: ModeBuild, SizeReport: true, SizeFormat: "json", DebugArtifactMode: DebugArtifactNone, DebugArtifactModeSet: true}
	for _, outputs := range [][2]io.Writer{{sizeFailWriter{}, io.Discard}, {io.Discard, sizeFailWriter{}}} {
		if err := reportBuildOutputs(conf, &OutFmtDetails{Out: path}, nil, outputs[0], outputs[1]); err == nil {
			t.Fatal("report write error was ignored")
		}
	}
}
