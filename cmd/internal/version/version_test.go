package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/gotool"
)

func TestVersionCommandHelper(t *testing.T) {
	if os.Getenv("LLGO_TEST_VERSION_COMMAND") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			runCmd(Cmd, os.Args[i+1:])
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func TestVersionInspectsGoFiles(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	goExe, err := gotool.Find(self, os.Getenv("PATH"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	program := filepath.Join(dir, "program with spaces")
	if err := os.WriteFile(source, []byte("package main\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(goExe, "build", "-o", program, source).CombinedOutput(); err != nil {
		t.Fatalf("build Go fixture: %v\n%s", err, output)
	}
	for _, tc := range []struct {
		args []string
		dir  string
	}{
		{args: []string{program}},
		{args: []string{"-m", program}},
		{args: []string{"-json", program}},
		{args: []string{"-v", source}},
		{args: []string{filepath.Join(dir, "missing")}},
		{args: []string{"-m"}},
		{args: []string{"-m=invalid", program}},
		{args: []string{"-unknown", program}},
		{args: []string{"-m", filepath.Base(program)}, dir: dir},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			expected := exec.Command(goExe, append([]string{"version"}, tc.args...)...)
			expected.Dir = tc.dir
			want, wantErr := expected.CombinedOutput()
			args := append([]string{}, tc.args...)
			if tc.dir != "" {
				args = append([]string{"-C", tc.dir}, args...)
			}
			actual := exec.Command(self, append([]string{"-test.run=^TestVersionCommandHelper$", "--"}, args...)...)
			actual.Env = append(os.Environ(), "LLGO_TEST_VERSION_COMMAND=1")
			got, gotErr := actual.CombinedOutput()
			if string(got) != string(want) || versionExitCode(gotErr) != versionExitCode(wantErr) {
				t.Fatalf("inspection differs from Go:\n got (%d) %s\nwant (%d) %s", versionExitCode(gotErr), got, versionExitCode(wantErr), want)
			}
		})
	}
	for _, args := range [][]string{nil, {"-v"}, {"-C", dir}} {
		command := exec.Command(self, append([]string{"-test.run=^TestVersionCommandHelper$", "--"}, args...)...)
		command.Env = append(os.Environ(), "LLGO_TEST_VERSION_COMMAND=1")
		output, err := command.CombinedOutput()
		if err != nil || !strings.HasPrefix(string(output), "llgo ") {
			t.Fatalf("LLGo identity %q = %s, %v", args, output, err)
		}
	}
}

func versionExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	return -1
}
