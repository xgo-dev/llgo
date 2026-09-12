//go:build !llgo
// +build !llgo

package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStaleEmscriptenGluePath(t *testing.T) {
	tests := []struct {
		driver string
		want   string
	}{
		{driver: "main.html", want: "main.mjs"},
		{driver: "main.js", want: "main.mjs"},
		{driver: "main.mjs", want: "main.js"},
		{driver: "main.wasm", want: ""},
		{driver: filepath.Join("out", "app.html"), want: filepath.Join("out", "app.mjs")},
	}
	for _, test := range tests {
		if got := staleEmscriptenGluePath(test.driver); got != test.want {
			t.Errorf("staleEmscriptenGluePath(%q) = %q, want %q", test.driver, got, test.want)
		}
	}
}

func TestRemoveStaleEmscriptenGlue(t *testing.T) {
	dir := t.TempDir()
	html := filepath.Join(dir, "main.html")
	stale := filepath.Join(dir, "main.mjs")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	conf := &Config{BuildMode: BuildModeExe, Target: "emscripten", Goos: "js"}
	if err := removeStaleEmscriptenGlue(conf, html); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale glue still present: %v", err)
	}
	if err := removeStaleEmscriptenGlue(conf, html); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveStaleEmscriptenGlueSkipsNativeOutputs(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "app.js")
	sibling := filepath.Join(dir, "app.mjs")
	if err := os.WriteFile(js, []byte("native"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleEmscriptenGlue(&Config{BuildMode: BuildModeExe, Goos: "darwin"}, js); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(sibling)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep" {
		t.Fatalf("native sibling glue = %q, want keep", got)
	}
}

func TestRemoveStaleEmscriptenGlueReportsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not make directories unwritable for the owner on Windows")
	}
	dir := t.TempDir()
	html := filepath.Join(dir, "main.html")
	stale := filepath.Join(dir, "main.mjs")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if err := removeStaleEmscriptenGlue(&Config{BuildMode: BuildModeExe, Target: "emscripten", Goos: "js"}, html); err == nil {
		t.Fatal("expected error removing stale glue from a read-only directory")
	}
}

func TestNeedsEmscriptenBrowserHost(t *testing.T) {
	tests := []struct {
		name   string
		conf   *Config
		output string
		want   bool
	}{
		{
			name:   "js html",
			conf:   &Config{BuildMode: BuildModeExe, Goos: "js"},
			output: "main.html",
			want:   true,
		},
		{
			name:   "emscripten html",
			conf:   &Config{BuildMode: BuildModeExe, Target: "emscripten"},
			output: "app.html",
			want:   true,
		},
		{
			name:   "emscripten-memory64 html",
			conf:   &Config{BuildMode: BuildModeExe, Target: "emscripten-memory64"},
			output: "app.html",
			want:   true,
		},
		{
			name:   "wasm alias html",
			conf:   &Config{BuildMode: BuildModeExe, Target: "wasm"},
			output: "app.html",
			want:   true,
		},
		{
			name:   "embedded html skips host shim",
			conf:   &Config{BuildMode: BuildModeExe, Target: "esp32"},
			output: "app.html",
		},
		{
			name:   "js js copies host shim",
			conf:   &Config{BuildMode: BuildModeExe, Goos: "js"},
			output: "main.js",
			want:   true,
		},
		{
			name:   "js mjs copies host shim",
			conf:   &Config{BuildMode: BuildModeExe, Goos: "js"},
			output: "main.mjs",
			want:   true,
		},
		{
			name:   "js wasm skips host shim",
			conf:   &Config{BuildMode: BuildModeExe, Goos: "js"},
			output: "main.wasm",
		},
		{
			name:   "archive skips html host",
			conf:   &Config{BuildMode: BuildModeCArchive, Goos: "js"},
			output: "main.html",
		},
		{
			name:   "nil config",
			output: "main.html",
		},
	}
	for _, test := range tests {
		if got := needsEmscriptenBrowserHost(test.conf, test.output); got != test.want {
			t.Errorf("%s: needsEmscriptenBrowserHost() = %v, want %v", test.name, got, test.want)
		}
	}
}

func TestInjectLLGoFSScript(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "main.html")
	const src = `<!doctype html><html><body><script type=module>import initModule from"./main.js";initModule(Module)</script></body></html>`
	if err := os.WriteFile(htmlPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := injectLLGoFSScript(htmlPath); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	want := `<script src="./wasm_fs.js"></script><script type=module>`
	if !strings.Contains(string(got), want) {
		t.Fatalf("injected HTML = %q, want to contain %q", got, want)
	}
	if err := injectLLGoFSScript(htmlPath); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(again), `<script src="./wasm_fs.js"></script>`) != 1 {
		t.Fatalf("repeated inject duplicated the script tag: %s", again)
	}
}

func TestInjectLLGoFSScriptQuotedModule(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "main.html")
	const src = `<html><script type='module' src="./main.js"></script></html>`
	if err := os.WriteFile(htmlPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := injectLLGoFSScript(htmlPath); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<script src="./wasm_fs.js"></script><script type='module'`) {
		t.Fatalf("HTML = %q", got)
	}
}

func TestInjectLLGoFSScriptErrors(t *testing.T) {
	if err := injectLLGoFSScript(filepath.Join(t.TempDir(), "missing.html")); err == nil {
		t.Fatal("missing HTML succeeded")
	}
	htmlPath := filepath.Join(t.TempDir(), "main.html")
	if err := os.WriteFile(htmlPath, []byte(`<html><body></body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := injectLLGoFSScript(htmlPath); err == nil || !strings.Contains(err.Error(), "no ES module script tag") {
		t.Fatalf("missing module tag error = %v", err)
	}
}

func TestInstallEmscriptenBrowserHost(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	htmlPath := filepath.Join(dir, "out", "main.html")
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const page = `<html><script type="module" src="./main.js"></script></html>`
	if err := os.WriteFile(htmlPath, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installEmscriptenBrowserHost(src, htmlPath); err != nil {
		t.Fatal(err)
	}
	if err := installEmscriptenBrowserHost(src, htmlPath); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(dir, "out", "wasm_fs.js")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "/* shim */\n" {
		t.Fatalf("copied wasm_fs.js = %q", data)
	}
	got, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<script src="./wasm_fs.js"></script><script type="module"`) {
		t.Fatalf("HTML = %q", got)
	}
}

func TestInstallEmscriptenBrowserHostJS(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsPath := filepath.Join(dir, "out", "main.js")
	if err := os.MkdirAll(filepath.Dir(jsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsPath, []byte("export default function Module() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installEmscriptenBrowserHost(src, jsPath); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(dir, "out", "wasm_fs.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != "/* shim */\n" {
		t.Fatalf("copied wasm_fs.js = %q", copied)
	}
	got, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "wasm_fs.js") {
		t.Fatalf("JS glue was modified: %q", got)
	}
}

func TestInstallEmscriptenBrowserHostErrors(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "out", "main.html")
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(htmlPath, []byte(`<html><body></body></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installEmscriptenBrowserHost(filepath.Join(dir, "missing.js"), htmlPath); err == nil {
		t.Fatal("missing source succeeded")
	}
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installEmscriptenBrowserHost(src, htmlPath); err == nil || !strings.Contains(err.Error(), "insert") {
		t.Fatalf("inject failure = %v", err)
	}
}

func TestInstallEmscriptenBrowserHostRejectsOutputCollision(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", wasmFSScriptName)
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", wasmFSScriptName)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("export default function Module() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := installEmscriptenBrowserHost(src, out)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("collision error = %v", err)
	}
	got, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "export default function Module() {}\n" {
		t.Fatalf("output was overwritten: %q", got)
	}
}

func TestCopyFileAtomicSamePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(path, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(path, path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "/* shim */\n" {
		t.Fatalf("same-path copy clobbered file: %q", got)
	}
}

func TestCopyFileAtomicMkdirAllError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(parent, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(src, filepath.Join(parent, "wasm_fs.js")); err == nil {
		t.Fatal("copyFileAtomic succeeded below a regular file")
	}
}

func TestCopyFileAtomicMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := copyFileAtomic(filepath.Join(dir, "missing.js"), filepath.Join(dir, "out", "wasm_fs.js")); err == nil {
		t.Fatal("copyFileAtomic succeeded with a missing source")
	}
}

func TestCopyFileAtomicDirectorySource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(src, filepath.Join(dir, "wasm_fs.js")); err == nil {
		t.Fatal("copyFileAtomic succeeded with a directory source")
	}
}

func TestCopyFileAtomicHardLink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "copy.js")
	if err := os.Link(src, dst); err != nil {
		t.Skipf("hard link: %v", err)
	}
	if err := copyFileAtomic(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "/* shim */\n" {
		t.Fatalf("hard-linked copy = %q", got)
	}
}

func TestCopyFileAtomicCreateTempError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory mode 0555 does not prevent CreateTemp on Windows")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := os.Mkdir(outDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outDir, 0o755) })
	err := copyFileAtomic(src, filepath.Join(outDir, "wasm_fs.js"))
	if err == nil {
		if os.Geteuid() == 0 {
			t.Skip("root can create files in a 0555 directory")
		}
		t.Fatal("copyFileAtomic succeeded in a read-only directory")
	}
}

func TestCopyFileAtomicOverwritesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* new */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.js")
	if err := os.WriteFile(dst, []byte("/* old */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "/* new */\n" {
		t.Fatalf("overwritten copy = %q", got)
	}
}

func TestReplaceFileExistingMissingAfterRename(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "missing", "out.js")
	if err := replaceFile(src, dst); err == nil {
		t.Fatal("replaceFile succeeded with a missing destination directory")
	}
}

func TestCopyFileAtomicDoesNotReplaceDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wasm_fs.js")
	if err := os.WriteFile(src, []byte("/* shim */\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "occupied")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(src, dst); err == nil {
		t.Fatal("copyFileAtomic replaced a directory")
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("copyFileAtomic replaced a directory with a file")
	}
}

func writeFakeLLGoRoot(t *testing.T, withShim bool) string {
	t.Helper()
	root := t.TempDir()
	runtimeDir := filepath.Join(root, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "go.mod"), []byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if withShim {
		targets := filepath.Join(root, "targets")
		if err := os.MkdirAll(targets, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(targets, wasmFSScriptName), []byte("/* shim */\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestPublishEmscriptenBrowserHost(t *testing.T) {
	if err := publishEmscriptenBrowserHost(nil, "main.html", false); err != nil {
		t.Fatalf("nil context: %v", err)
	}
	native := &context{buildConf: &Config{BuildMode: BuildModeExe, Goos: "darwin"}}
	if err := publishEmscriptenBrowserHost(native, "main.html", false); err != nil {
		t.Fatalf("native target: %v", err)
	}
	js := &context{buildConf: &Config{BuildMode: BuildModeExe, Goos: "js"}}
	if err := publishEmscriptenBrowserHostFrom(js, "main.html", false, ""); err == nil {
		t.Fatal("empty LLGO_ROOT succeeded")
	}

	root := writeFakeLLGoRoot(t, true)
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "main.html")
	if err := os.WriteFile(htmlPath, []byte(`<html><script type=module>import initModule from"./main.js"</script></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{BuildMode: BuildModeExe, Goos: "js", PrintCommands: true}}
	if err := publishEmscriptenBrowserHost(ctx, htmlPath, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, wasmFSScriptName)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<script src="./wasm_fs.js"></script>`) {
		t.Fatalf("HTML = %q", got)
	}

	missingRoot := writeFakeLLGoRoot(t, false)
	t.Setenv("LLGO_ROOT", missingRoot)
	if err := publishEmscriptenBrowserHost(ctx, htmlPath, false); err == nil {
		t.Fatal("missing shim succeeded")
	}
}
