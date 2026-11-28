package build

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestExecHelperProcess(t *testing.T) {
	if os.Getenv("LLGO_TEST_EXEC_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg != "--" {
			continue
		}
		args := os.Args[i+1:]
		switch args[0] {
		case "args":
			cwd, _ := os.Getwd()
			_ = json.NewEncoder(os.Stdout).Encode(struct {
				Args []string
				Dir  string
				PWD  string
			}{args[1:], cwd, os.Getenv("PWD")})
		case "exit":
			os.Exit(23)
		case "hang":
			time.Sleep(time.Minute)
		}
		os.Exit(0)
	}
	os.Exit(99)
}

func TestExecOverridesTargetRunner(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	commands := commandEnv{dir: dir, environ: append(os.Environ(), "LLGO_TEST_EXEC_HELPER=1")}
	for _, artifact := range []string{"program with spaces", "module {out}.wasm", "firmware.elf"} {
		program := testProgram{app: filepath.Join(dir, artifact), pkgDir: dir, runner: "missing-default-runner", pkgName: "example.com/fixture"}
		conf := &Config{
			Exec:          []string{self, "-test.run=^TestExecHelperProcess$", "--", "args"},
			RunArgs:       []string{"argument with spaces", "{out}", "-guest-flag", "semi;colon"},
			Target:        "named-target",
			PrintCommands: true,
		}
		var stdout, stderr bytes.Buffer
		if err := runNativeTest(commands, program, conf, &stdout, &stderr); err != nil {
			t.Fatalf("custom runner: %v\n%s", err, &stderr)
		}
		wantTrace := fmt.Sprintf("%s -test.run=^TestExecHelperProcess$ -- args %s argument with spaces {out} -guest-flag semi;colon\n", self, program.app)
		if stderr.String() != wantTrace {
			t.Fatalf("wrapper command trace = %q, want %q", stderr.String(), wantTrace)
		}
		var result struct {
			Args []string
			Dir  string
			PWD  string
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := append([]string{program.app}, conf.RunArgs...)
		if !reflect.DeepEqual(result.Args, want) {
			t.Fatalf("wrapper arguments = %q, want %q", result.Args, want)
		}
		gotDir, _ := filepath.EvalSymlinks(result.Dir)
		wantDir, _ := filepath.EvalSymlinks(dir)
		if gotDir != wantDir {
			t.Fatalf("wrapper directory = %q, want %q", result.Dir, dir)
		}
		pwd, _ := filepath.EvalSymlinks(result.PWD)
		if pwd != wantDir {
			t.Fatalf("wrapper PWD = %q, want %q", result.PWD, dir)
		}
	}
}

func TestExecExitAndTimeout(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := commandEnv{environ: append(os.Environ(), "LLGO_TEST_EXEC_HELPER=1")}
	conf := &Config{Exec: []string{self, "-test.run=^TestExecHelperProcess$", "--", "exit"}}
	err = runWithExec(commands, "program", conf, runnerDetails{phase: "run"}, new(bytes.Buffer), new(bytes.Buffer))
	if code, ok := RunnerExitCode(err); !ok || code != 23 {
		t.Fatalf("wrapper exit = %v, want status 23", err)
	}
	conf.Exec[len(conf.Exec)-1] = "hang"
	conf.RunnerTimeout = 100 * time.Millisecond
	err = runNativeTest(commands, testProgram{app: "program"}, conf, new(bytes.Buffer), new(bytes.Buffer))
	var failure *runnerFailure
	if !errors.As(err, &failure) || failure.status != runnerStatusTimeout {
		t.Fatalf("wrapper timeout = %v, want timed-out failure", err)
	}
}
