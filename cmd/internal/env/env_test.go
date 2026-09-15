//go:build !llgo

package env

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestExtendedQuery(t *testing.T) {
	for _, test := range []struct {
		args []string
		want bool
	}{
		{nil, true},
		{[]string{"-json"}, true},
		{[]string{"-json=true"}, true},
		{[]string{"-changed"}, true},
		{[]string{"GOOS", "GOARCH"}, false},
		{[]string{"-json", "GOOS"}, false},
		{[]string{"GOOS", "LLGO_ROOT"}, true},
		{[]string{"-target", "rp2040", "GOOS"}, true},
		{[]string{"-invalid-env-flag"}, false},
		{[]string{"-w", "GOPROXY=https://example.invalid"}, false},
		{[]string{"-w", "LLGO_ROOT=/tmp/llgo"}, true},
	} {
		if got := extendedQuery(test.args); got != test.want {
			t.Errorf("extendedQuery(%q) = %v, want %v", test.args, got, test.want)
		}
	}
}

func TestEnvMatchesGo(t *testing.T) {
	t.Setenv("GOENV", filepath.Join(t.TempDir(), "goenv"))
	t.Setenv("GOTOOLCHAIN", "local")
	for _, args := range [][]string{
		{"GOOS", "GOARCH"},
		{"-json", "GOOS", "GOARCH", "GOROOT", "GOVERSION"},
		{"-changed", "GOOS", "GOARCH"},
		{"LLGO_UNKNOWN_ENV_VARIABLE"},
		{"-invalid-env-flag"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("GOOS", "wasip1")
			t.Setenv("GOARCH", "wasm")
			var wantOut, wantErr, gotOut, gotErr bytes.Buffer
			cmd := exec.Command("go", append([]string{"env"}, args...)...)
			cmd.Stdout, cmd.Stderr = &wantOut, &wantErr
			want := cmd.Run()
			got := run(args, nil, &gotOut, &gotErr)
			if (want == nil) != (got == nil) || wantOut.String() != gotOut.String() || wantErr.String() != gotErr.String() {
				t.Fatalf("env %q: got (%q, %q, %v), want (%q, %q, %v)", args, &gotOut, &gotErr, got, &wantOut, &wantErr, want)
			}
		})
	}
	var out bytes.Buffer
	if err := run(nil, nil, &out, &out); err != nil || !strings.Contains(out.String(), "GOARCH=") {
		t.Fatalf("env without args: %v, %s", err, &out)
	}
}

func TestEnvSettings(t *testing.T) {
	config := filepath.Join(t.TempDir(), "goenv")
	t.Setenv("GOENV", config)
	t.Setenv("GOTOOLCHAIN", "local")
	for _, args := range [][]string{{"-w", "GOPROXY=https://example.invalid"}, {"-u", "GOPROXY"}} {
		var output bytes.Buffer
		if err := run(args, nil, &output, &output); err != nil {
			t.Fatalf("env %q: %v, %s", args, err, &output)
		}
		data, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "GOPROXY=") != (args[0] == "-w") {
			t.Fatalf("env %q wrote %q", args, data)
		}
	}
}

func TestLLGoEnvironment(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(name, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("runtime/go.mod", "module github.com/xgo-dev/llgo/runtime\n")
	mustWrite("targets/base.json", `{"goos":"linux","goarch":"arm","libc":"picolibc","serial":"usb"}`)
	mustWrite("targets/board.json", `{"inherits":["base"],"cpu":"cortex-m0plus","serial-port":["cafe:babe"]}`)
	if err := os.MkdirAll(filepath.Join(root, "crosscompile", "clang"), 0755); err != nil {
		t.Fatal(err)
	}

	cacheHome := t.TempDir()
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("LLVM_CONFIG", filepath.Join(root, "missing-llvm-config"))
	t.Setenv("HOME", cacheHome)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheHome, "cache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(cacheHome, "cache"))

	var output bytes.Buffer
	args := []string{"-json", "-target", "board", "GOOS", "LLGO_ROOT", "LLGO_RUNTIME_DIR", "LLGO_EMBEDDED_CLANG_DIR", "LLGO_WASI_LIBC_DIR", "LLGO_TARGET_GOOS", "LLGO_TARGET_GOARCH", "LLGO_TARGET_CPU", "LLGO_TARGET_LIBC", "LLGO_TARGET_SERIAL", "LLGO_TARGET_SERIAL_PORTS"}
	if err := run(args, nil, &output, &output); err != nil {
		t.Fatalf("env %q: %v, %s", args, err, &output)
	}
	var got map[string]string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"GOOS":                     runtime.GOOS,
		"LLGO_ROOT":                root,
		"LLGO_RUNTIME_DIR":         filepath.Join(root, "runtime"),
		"LLGO_EMBEDDED_CLANG_DIR":  filepath.Join(root, "crosscompile", "clang"),
		"LLGO_WASI_LIBC_DIR":       "",
		"LLGO_TARGET_GOOS":         "linux",
		"LLGO_TARGET_GOARCH":       "arm",
		"LLGO_TARGET_CPU":          "cortex-m0plus",
		"LLGO_TARGET_LIBC":         "picolibc",
		"LLGO_TARGET_SERIAL":       "usb",
		"LLGO_TARGET_SERIAL_PORTS": "cafe:babe",
	} {
		if got[name] != want {
			t.Errorf("%s = %q, want %q", name, got[name], want)
		}
	}
	if _, err := os.Stat(filepath.Join(cacheHome, "cache", "llgo", "crosscompile")); !os.IsNotExist(err) {
		t.Fatalf("env inspection created the cross-compilation cache: %v", err)
	}

	output.Reset()
	if err := run([]string{"-json", "GOARCH", "LLGO_LLVM_CONFIG", "LLGO_LLVM_VERSION"}, nil, &output, &output); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["LLGO_LLVM_CONFIG"] != filepath.Join(root, "missing-llvm-config") || got["LLGO_LLVM_VERSION"] != "" {
		t.Fatalf("LLVM diagnostics = %#v", got)
	}

	if err := run([]string{"-w", "LLGO_ROOT=" + root}, nil, &output, &output); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("writing LLGO_ROOT: %v", err)
	}
	if err := run([]string{"-target", "missing", "LLGO_TARGET"}, nil, &output, &output); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing target: %v", err)
	}
}

func TestWasmTargetEnvironment(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, test := range []struct {
		target, profile, provider, goos, goarch, llvmTarget string
	}{
		{"emscripten", "j32", "emscripten", "js", "wasm", "wasm32-unknown-emscripten"},
		{"emscripten-memory64", "j64", "emscripten", "js", "wasm", "wasm64-unknown-emscripten"},
		{"wasi", "w32", "wasi", "wasip1", "wasm", "wasm32-unknown-wasip1"},
		{"wasm", "j32", "emscripten", "js", "wasm", "wasm32-unknown-emscripten"},
		{"wasip1", "w32", "wasi", "wasip1", "wasm", "wasm32-unknown-wasip1"},
		{"cortex-m0", "", "", "linux", "arm", "thumbv6m-unknown-unknown-eabi"},
	} {
		t.Run(test.target, func(t *testing.T) {
			var output bytes.Buffer
			args := []string{"-json", "-target", test.target, "LLGO_TARGET", "LLGO_TARGET_WASM_PROFILE", "LLGO_TARGET_WASM_PROVIDER", "LLGO_TARGET_GOOS", "LLGO_TARGET_GOARCH", "LLGO_TARGET_LLVM_TARGET"}
			if err := run(args, nil, &output, &output); err != nil {
				t.Fatalf("env %q: %v, %s", args, err, &output)
			}
			var got map[string]string
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string]string{
				"LLGO_TARGET":               test.target,
				"LLGO_TARGET_WASM_PROFILE":  test.profile,
				"LLGO_TARGET_WASM_PROVIDER": test.provider,
				"LLGO_TARGET_GOOS":          test.goos,
				"LLGO_TARGET_GOARCH":        test.goarch,
				"LLGO_TARGET_LLVM_TARGET":   test.llvmTarget,
			} {
				if value, ok := got[name]; !ok || value != want {
					t.Errorf("%s = %q (present %v), want %q", name, value, ok, want)
				}
			}
		})
	}
}

func TestExtendedFormattingAndHelpers(t *testing.T) {
	t.Setenv("LLGO_ROOT", "")
	t.Setenv("LLVM_CONFIG", filepath.Join(t.TempDir(), "missing-llvm-config"))

	for _, args := range [][]string{{"-h", "LLGO_ROOT"}, {"-bad", "LLGO_ROOT"}} {
		var output bytes.Buffer
		err := run(args, nil, &output, &output)
		if (err == nil) != (args[0] == "-h") {
			t.Errorf("env %q error = %v, output %q", args, err, &output)
		}
	}
	var output bytes.Buffer
	if err := run([]string{"LLGO_ROOT", "LLGO_RUNTIME_PKG"}, nil, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.ReplaceAll(output.String(), "\r\n", "\n"), "\ngithub.com/xgo-dev/llgo/runtime\n") {
		t.Fatalf("explicit output = %q", &output)
	}
	output.Reset()
	if err := run([]string{"-changed", "LLGO_VERSION"}, nil, &output, &output); err != nil || output.Len() != 0 {
		t.Fatalf("unchanged output = %q, %v", &output, err)
	}
	if err := run([]string{"-target", "board", "-w", "GOOS=linux"}, nil, &output, &output); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("target write error = %v", err)
	}

	dir := t.TempDir()
	name := "clang"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	clang := filepath.Join(dir, name)
	if err := os.WriteFile(clang, nil, 0755); err != nil {
		t.Fatal(err)
	}
	info := &llvmInfo{
		config: "unused",
		fields: map[string]func() string{"--bindir": func() string { return dir }},
		tools:  make(map[string]func() string),
	}
	if got := info.tool("clang"); got != clang {
		t.Errorf("tool = %q, want %q", got, clang)
	}
	info.fields["--bindir"] = func() string { return filepath.Join(dir, "missing") }
	if got := info.tool("go"); got == "" {
		t.Error("PATH fallback did not find go")
	}
	if got := commandLine("go", "version"); !strings.HasPrefix(got, "go version") {
		t.Errorf("commandLine(go version) = %q", got)
	}
	if got := commandLine(filepath.Join(dir, "missing"), "--version"); got != "" {
		t.Errorf("missing command = %q", got)
	}
	if got := existingDir(""); got != "" {
		t.Errorf("existingDir(empty) = %q", got)
	}
	if got := existingDir(dir); got != dir {
		t.Errorf("existingDir = %q", got)
	}
	if got := joinIfSet(""); got != "" {
		t.Errorf("joinIfSet(empty) = %q", got)
	}
	if got := joinIfSet(dir, "a", "b"); got != filepath.Join(dir, "a", "b") {
		t.Errorf("joinIfSet = %q", got)
	}
	if got := firstWord("  qemu-system-arm -M board"); got != "qemu-system-arm" {
		t.Errorf("firstWord = %q", got)
	}
	if got := firstWord(""); got != "" {
		t.Errorf("firstWord(empty) = %q", got)
	}
	if got := resolveFirstTool([]string{"missing-llgo-tool", "go"}, ""); got == "" {
		t.Error("resolveFirstTool did not try its fallback")
	}
	if got := resolveTool(clang, ""); got != clang {
		t.Errorf("resolveTool(absolute) = %q", got)
	}
	if got := resolveTool("missing-llgo-tool", dir); got != "" {
		t.Errorf("resolveTool(missing) = %q", got)
	}
	if got := librarySourceDir(dir, "lib-version"); got != filepath.Join(dir, "crosscompile", "lib-version") {
		t.Errorf("librarySourceDir = %q", got)
	}
	if got := envAssignment("linux", "X", "a'b"); got != `X='a'\''b'` {
		t.Errorf("Unix assignment = %q", got)
	}
	if got := envAssignment("windows", "X", "%a&b^c\n"); got != "set X=%%a^&b^^c�" {
		t.Errorf("Windows assignment = %q", got)
	}
	if got := envAssignment("linux", "X", "safe\x1b[31m\nline"); got != "X='safe�[31m�line'" {
		t.Errorf("sanitized Unix assignment = %q", got)
	}
	windowsCandidates := toolCandidates("windows", `C:\LLVM\bin`, "ld.lld")
	if len(windowsCandidates) != 2 || !strings.HasSuffix(windowsCandidates[1], "ld.lld.exe") {
		t.Errorf("Windows dotted tool candidates = %q", windowsCandidates)
	}
	if candidates := toolCandidates("windows", `C:\LLVM\bin`, "clang.exe"); len(candidates) != 1 {
		t.Errorf("Windows exe candidates = %q", candidates)
	}
}

func TestRunCmdExitStatus(t *testing.T) {
	for _, test := range []struct {
		name      string
		args      []string
		missingGo bool
		code      int
	}{
		{name: "Go diagnostic", args: []string{"-invalid-env-flag"}, code: 2},
		{name: "missing Go", missingGo: true, code: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.missingGo {
				t.Setenv("PATH", t.TempDir())
			}
			mockable.EnableMock()
			defer mockable.DisableMock()
			defer func() {
				if got := recover(); got != "exit" || mockable.ExitCode() != test.code {
					t.Errorf("exit = %v, %d; want %d", got, mockable.ExitCode(), test.code)
				}
			}()
			Cmd.Run(Cmd, test.args)
		})
	}
}
