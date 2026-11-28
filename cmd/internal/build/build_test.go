package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/mockable"
	cmdflags "github.com/xgo-dev/llgo/cmd/internal/flags"
)

func TestRunCmdPassesGoBuildFlags(t *testing.T) {
	oldArgs := append([]string(nil), goBuildFlags.Args...)
	goBuildFlags.Args = nil
	defer func() { goBuildFlags.Args = oldArgs }()

	stderrFile, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderrFile
	defer func() { os.Stderr = oldStderr }()

	mockable.EnableMock()
	defer mockable.DisableMock()
	exited := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == "exit" {
					exited = true
					return
				}
				panic(recovered)
			}
		}()
		runCmd(Cmd, []string{"-gcflags=all=-N", "-ldflags=-s -w", filepath.Join(t.TempDir(), "missing")})
	}()
	if !exited || mockable.ExitCode() != 1 {
		t.Fatalf("runCmd exit = (%v, %d), want (true, 1)", exited, mockable.ExitCode())
	}
	if !reflect.DeepEqual(goBuildFlags.Args, []string{"-gcflags=all=-N", "-ldflags=-s -w"}) {
		t.Fatalf("go build flags = %v", goBuildFlags.Args)
	}
	if err := stderrFile.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(stderrFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "missing") {
		t.Fatalf("stderr = %q, want missing-package diagnostic", data)
	}
}

func TestBuildCommandHasSchedulerTraceFlag(t *testing.T) {
	flag := Cmd.Flag.Lookup("debug-trace")
	if flag == nil {
		t.Fatal("llgo build has no -debug-trace flag")
	}
	if !strings.Contains(flag.Usage, "Chrome/Perfetto") {
		t.Fatalf("-debug-trace usage = %q", flag.Usage)
	}
}

func TestBuildCommandCoverageFlags(t *testing.T) {
	for _, name := range []string{"cover", "covermode", "coverpkg"} {
		if Cmd.Flag.Lookup(name) == nil {
			t.Fatalf("llgo build has no -%s flag", name)
		}
	}
	if Cmd.Flag.Lookup("coverprofile") != nil {
		t.Fatal("llgo build exposes test-only -coverprofile")
	}
}

func TestRunCmdBuildsMultiplePackagesToDirectory(t *testing.T) {
	root := t.TempDir()
	for name, output := range map[string]string{"first": "first", "second": "second"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		source := "package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"" + output + "\") }\n"
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/cmdmultibuild\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "bin")

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
	oldOutput := cmdflags.OutputFile
	oldPassArgs := append([]string(nil), goBuildFlags.Args...)
	defer func() {
		cmdflags.OutputFile = oldOutput
		_ = Cmd.Flag.Set("o", oldOutput)
		goBuildFlags.Args = oldPassArgs
	}()
	cmdflags.OutputFile = ""
	goBuildFlags.Args = nil
	runCmd(Cmd, []string{"-o", out + string(os.PathSeparator), "./..."})

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	for _, name := range []string{"first", "second"} {
		data, err := exec.Command(filepath.Join(out, name+ext)).CombinedOutput()
		if err != nil || strings.TrimSpace(string(data)) != name {
			t.Fatalf("run %s: %v, output %q", name, err, data)
		}
	}
}
