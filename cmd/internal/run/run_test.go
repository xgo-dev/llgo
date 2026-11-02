package run

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestParseRunArgs(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		files   []string
		program []string
		wantErr bool
	}{
		{args: []string{"main.go", "helper.go", "-flag", "literal.go"}, files: []string{"main.go", "helper.go"}, program: []string{"-flag", "literal.go"}},
		{args: []string{".", "argument.go"}, files: []string{"."}, program: []string{"argument.go"}},
		{args: []string{"path with spaces/main.go", "path with spaces/helper.go"}, files: []string{"path with spaces/main.go", "path with spaces/helper.go"}, program: []string{}},
		{args: []string{"main.go", "unit_test.go"}, wantErr: true},
		{args: []string{"example.com/tool@v1.2.3"}, wantErr: true},
		{args: []string{"main@version.go"}, files: []string{"main@version.go"}, program: []string{}},
		{wantErr: true},
	} {
		files, program, err := parseRunArgs(tc.args)
		if (err != nil) != tc.wantErr || !reflect.DeepEqual(files, tc.files) || !reflect.DeepEqual(program, tc.program) {
			t.Errorf("parseRunArgs(%q) = (%q, %q, %v), want (%q, %q), error=%v", tc.args, files, program, err, tc.files, tc.program, tc.wantErr)
		}
	}
}

func TestRunMultipleFilesPreservesProgramArgs(t *testing.T) {
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	dir := t.TempDir()
	main := filepath.Join(dir, "main.go")
	helper := filepath.Join(dir, "helper.go")
	out := filepath.Join(dir, "output with spaces")
	for path, source := range map[string]string{
		main: `package main
import ("os"; "strings")
func main() { if err := os.WriteFile(os.Args[1], []byte(helper()+"|"+strings.Join(os.Args[2:], "|")), 0600); err != nil { panic(err) } }
`,
		helper: "package main\nfunc helper() string { return \"second file\" }\n",
	} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	mockable.EnableMock()
	defer mockable.DisableMock()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "wrapper arguments.json")
	quote := func(s string) string { return `"` + s + `"` }
	wrapper := strings.Join([]string{quote(self), "-test.run=^TestRunExecHelper$", "--", quote(log)}, " ")
	oldExec := *runExec
	defer func() { *runExec = oldExec }()
	t.Setenv("LLGO_TEST_RUN_EXEC", "1")
	runCmd(Cmd, []string{"-exec=" + wrapper, main, helper, out, "-flag", "literal.go"})
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "second file|-flag|literal.go" {
		t.Fatalf("multi-file run output = %q, %v", data, err)
	}
	data, err = os.ReadFile(log)
	var args []string
	if err != nil || json.Unmarshal(data, &args) != nil || len(args) != 4 || !reflect.DeepEqual(args[1:], []string{out, "-flag", "literal.go"}) {
		t.Fatalf("wrapper did not receive the linked artifact and program arguments: %q, %v", data, err)
	}
}

func TestCmpTestUsesSelectedFiles(t *testing.T) {
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "")
	dir := t.TempDir()
	for name, source := range map[string]string{
		"main.go":       "package main\nimport \"fmt\"\nfunc main() { fmt.Println(helper()) }\n",
		"helper.go":     "package main\nfunc helper() string { return \"selected files\" }\n",
		"unselected.go": "package main\nfunc init() { panic(\"unselected file compiled\") }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	mockable.EnableMock()
	defer mockable.DisableMock()
	runCmpTest(CmpTestCmd, []string{filepath.Join(dir, "main.go"), filepath.Join(dir, "helper.go")})
}

func TestRunExecHelper(t *testing.T) {
	if os.Getenv("LLGO_TEST_RUN_EXEC") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			args := os.Args[i+1:]
			data, _ := json.Marshal(args[1:])
			if err := os.WriteFile(args[0], data, 0600); err != nil {
				os.Exit(98)
			}
			command := exec.Command(args[1], args[2:]...)
			command.Stdout, command.Stderr, command.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := command.Run(); err != nil {
				if exit, ok := err.(*exec.ExitError); ok {
					os.Exit(exit.ExitCode())
				}
				os.Exit(97)
			}
			os.Exit(0)
		}
	}
	os.Exit(99)
}

func TestRunAndCmpTestBuildFlagsAreIndependent(t *testing.T) {
	if runGoBuildFlags == cmpTestGoBuildFlags {
		t.Fatal("run and cmptest share a build-flag collector")
	}
	if Cmd.Flag.Lookup("ldflags") == nil || CmpTestCmd.Flag.Lookup("ldflags") == nil {
		t.Fatal("run or cmptest does not register -ldflags")
	}
	if Cmd.Flag.Lookup("timeout") == nil {
		t.Fatal("run does not register -timeout")
	}
	if CmpTestCmd.Flag.Lookup("timeout") != nil {
		t.Fatal("cmptest unexpectedly registers the run-only -timeout")
	}
	if runGoBuildFlags.Flag != &Cmd.Flag || cmpTestGoBuildFlags.Flag != &CmpTestCmd.Flag {
		t.Fatal("run or cmptest build flags are bound to the wrong command")
	}
}

func TestRunPreservesNativeExitCode(t *testing.T) {
	program := filepath.Join(t.TempDir(), "exit.go")
	if err := os.WriteFile(program, []byte("package main\nimport \"os\"\nfunc main() { os.Exit(23) }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer func() {
		if recovered := recover(); recovered != "exit" || mockable.ExitCode() != 23 {
			t.Errorf("llgo run exit = (%v, %d), want (exit, 23)", recovered, mockable.ExitCode())
		}
	}()
	runCmd(Cmd, []string{program})
}
