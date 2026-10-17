package llar

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeLLAR(t *testing.T, output string) (bin, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	bin = filepath.Join(dir, "llar")
	argsFile = filepath.Join(dir, "args")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"$LLAR_TEST_ARGS\"\n" +
		"printf '%s\\n' 'progress' >&2\n" +
		"printf '%s\\n' '" + output + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLAR_TEST_ARGS", argsFile)
	return bin, argsFile
}

func readArgs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestInstall(t *testing.T) {
	bin, argsFile := writeFakeLLAR(t, `{"path":"owner/root","version":"v1.2.3","dir":"/tmp/root","deps":[{"path":"owner/dep","version":"v1.0.0","dir":"/tmp/dep"}],"metadata":"-L/tmp/root/lib -lroot"}`)

	var rawStdout, rawStderr bytes.Buffer
	cmd := New(bin)
	cmd.Stdout = &rawStdout
	cmd.Stderr = &rawStderr
	result, err := cmd.Install(Module{Path: "owner/root", Version: "v1.2.3"}, Config{
		To:   "/tmp/root",
		OS:   "linux",
		Arch: "amd64",
		Libc: "glibc",
		Options: map[string][]string{
			"zlib":  {"system", "bundled"},
			"debug": {"true"},
		},
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Path != "owner/root" || result.Version != "v1.2.3" || result.Dir != "/tmp/root" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Deps) != 1 || result.Deps[0].Path != "owner/dep" {
		t.Fatalf("deps = %+v", result.Deps)
	}
	if result.BuildFlags != "-L/tmp/root/lib -lroot" {
		t.Fatalf("build flags = %q", result.BuildFlags)
	}
	if !strings.Contains(rawStdout.String(), `"path":"owner/root"`) {
		t.Fatalf("stdout = %q, want JSON result", rawStdout.String())
	}
	if rawStderr.String() != "progress\n" {
		t.Fatalf("stderr = %q, want progress output", rawStderr.String())
	}

	wantArgs := []string{
		"install", "--json", "--output", "/tmp/root",
		"--verbose",
		"--os", "linux", "--arch", "amd64", "--libc", "glibc",
		"--option", "debug=true",
		"--option", "zlib=system",
		"--option", "zlib=bundled",
		"owner/root@v1.2.3",
	}
	gotArgs := readArgs(t, argsFile)
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %q, want %q", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q, want %q", i, gotArgs[i], wantArgs[i])
		}
	}
}

func TestInstallWithoutOutput(t *testing.T) {
	bin, argsFile := writeFakeLLAR(t, `{"path":"owner/root","version":"v1.0.0","metadata":"-lroot"}`)
	if _, err := New(bin).Install(Module{Path: "owner/root", Version: "v1.0.0"}, Config{}); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	want := []string{"install", "--json", "owner/root@v1.0.0"}
	got := readArgs(t, argsFile)
	if len(got) != len(want) {
		t.Fatalf("args = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestInstallReturnsCommandError(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "llar")
	script := "#!/bin/sh\nprintf '%s\\n' 'install failed' >&2\nexit 7\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := New(bin).Install(Module{Path: "owner/root"}, Config{To: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "install failed") || !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("error = %v", err)
	}
}

func TestInstallReturnsJSONError(t *testing.T) {
	bin, _ := writeFakeLLAR(t, "not-json")
	_, err := New(bin).Install(Module{Path: "owner/root"}, Config{})
	if err == nil || !strings.Contains(err.Error(), "decode result") {
		t.Fatalf("error = %v", err)
	}
}

func TestResultJSONRoundTrip(t *testing.T) {
	want := Result{
		Path: "owner/root", Version: "v1.0.0", Dir: "/tmp/root",
		Deps:       []Dependency{{Path: "owner/dep", Version: "v1.0.0", Dir: "/tmp/dep"}},
		BuildFlags: "-lroot",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Path != want.Path || got.Version != want.Version || got.Dir != want.Dir || got.BuildFlags != want.BuildFlags || len(got.Deps) != 1 {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}
