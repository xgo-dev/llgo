package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Some Go runtime contracts deliberately terminate the process. WebAssembly
// guests cannot spawn a child, so the acceptance host executes the exact test
// artifact a second time with a private selector argument.
func fullChildCommand(p profile, root, goRoot, artifact, arg string) command {
	env := map[string]string{"GOMAXPROCS": "1"}
	var args []string
	switch {
	case p.Reference:
		args = []string{filepath.Join(goRoot, "lib", "wasm", "go_"+p.GOOS+"_wasm_exec"), artifact, arg}
		env["GOWASIRUNTIME"] = "wasmtime"
	case p.Target == "wasi":
		args = []string{"wasmtime", "run", "-W", "exceptions=y", "--dir=" + root, artifact, arg}
	default:
		runner := "emscripten-runner.mjs"
		if p.Target == "emscripten-memory64" {
			runner = "emscripten-memory64-runner.mjs"
		}
		args = []string{"node", filepath.Join(root, "targets", runner), artifact, arg}
	}
	return command{"timeout", append([]string{"--kill-after=10s", "30s"}, args...), env}
}

func fullPanicCommand(p profile, root, goRoot, artifact string) command {
	return fullChildCommand(p, root, goRoot, artifact, "-llgo.caller-panic-child")
}

var fullFinalizerInvalidCases = []string{"non-function", "no parameters", "two parameters", "variadic", "wrong type"}

func fullFinalizerInvalidCommand(p profile, root, goRoot, artifact, name string) command {
	return fullChildCommand(p, root, goRoot, artifact, "-llgo.finalizer-invalid-case="+name)
}

func fullBuiltinPrintCommand(p profile, root, goRoot, artifact string) command {
	return fullChildCommand(p, root, goRoot, artifact, "-llgo.builtin-print-child")
}

func fullGoexitLifecycleCommand(p profile, root, goRoot, artifact string) command {
	return fullChildCommand(p, root, goRoot, artifact, "-llgo.main-goexit-lifecycle-child")
}

const fullBuiltinPrintWant = "1e+07\n(1e+07-1e+07i)\n(1.5+0i)\nNaN\n+Inf\n-Inf\n(1+NaNi)\n(1+Infi)\n(1-Infi)\n"

func validateFullBuiltinPrint(out []byte, runErr error) error {
	if runErr != nil {
		return fmt.Errorf("builtin-print child failed: %w", runErr)
	}
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != fullBuiltinPrintWant {
		return fmt.Errorf("builtin-print output = %q, want %q", got, fullBuiltinPrintWant)
	}
	return nil
}

func validateFullGoexitLifecycle(out []byte, runErr error) error {
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) || exit.ExitCode() < 1 || exit.ExitCode() > 2 {
		return fmt.Errorf("Goexit lifecycle child must exit 1 or 2, not succeed, time out, or fail to launch: %v", runErr)
	}
	worker := strings.Index(string(out), "WORKER_RETURNING")
	deadlock := strings.Index(string(out), "no goroutines (main called runtime.Goexit) - deadlock!")
	if worker < 0 || deadlock < 0 || worker > deadlock {
		return errors.New("Goexit lifecycle child did not release its worker before reporting the last-goroutine deadlock")
	}
	return nil
}

func validateFullFinalizerInvalid(out []byte, runErr error, name string) error {
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) || exit.ExitCode() < 1 || exit.ExitCode() > 2 {
		return fmt.Errorf("invalid-finalizer child must exit 1 or 2, not succeed, time out, or fail to launch: %v", runErr)
	}
	want := "runtime.SetFinalizer:"
	switch name {
	case "non-function":
		want += " second argument is"
	case "variadic":
		want += " cannot pass"
		if !strings.Contains(string(out), "because dotdotdot") {
			return fmt.Errorf("invalid-finalizer %s diagnostic is missing %q", name, "because dotdotdot")
		}
	case "no parameters", "two parameters", "wrong type":
		want += " cannot pass"
	default:
		return fmt.Errorf("unknown invalid-finalizer case %q", name)
	}
	if !strings.Contains(string(out), want) {
		return fmt.Errorf("invalid-finalizer %s diagnostic is missing %q", name, want)
	}
	return nil
}

func validateFullPanic(root string, out []byte, runErr error) error {
	var exit *exec.ExitError
	if !errors.As(runErr, &exit) || exit.ExitCode() < 1 || exit.ExitCode() > 2 {
		return fmt.Errorf("panic child must exit 1 or 2, not succeed, time out, or fail to launch: %v", runErr)
	}
	source, err := os.ReadFile(filepath.Join(root, "test", "go", "caller_runtime_test.go"))
	if err != nil {
		return err
	}
	for _, marker := range []string{"PANIC_MARK", "PANIC_CALLER_MARK"} {
		line := 0
		for i, text := range strings.Split(string(source), "\n") {
			if strings.HasSuffix(strings.TrimSpace(text), "// "+marker) {
				line = i + 1
				break
			}
		}
		if line == 0 {
			return fmt.Errorf("missing panic source marker %s", marker)
		}
		location := fmt.Sprintf("caller_runtime_test.go:%d", line)
		if !regexp.MustCompile(regexp.QuoteMeta(location) + `(?:\D|$)`).Match(out) {
			return fmt.Errorf("panic child traceback missing exact source location %q", location)
		}
	}
	for _, text := range []string{"panic: acceptance-boom", "goroutine 1 [running]:", "callerPanicBoom", "callerPanicCaller"} {
		if !strings.Contains(string(out), text) {
			return fmt.Errorf("panic child traceback missing %q", text)
		}
	}
	return nil
}

type fullHostValidator func([]byte, error) error

func runFullHostCheck(root, reportPath, profileName, packageName, logName, checkName string, cmd command, e *fullPackage, validate fullHostValidator, run func(string, command) ([]byte, error)) error {
	out, runErr := run(root, cmd)
	if err := os.WriteFile(filepath.Join(reportPath+".logs", logName), out, 0o644); err != nil {
		return err
	}
	check := fullHostCheck{Name: checkName, Status: "pass"}
	if err := validate(out, runErr); err != nil {
		check.Status, check.Reason = "fail", err.Error()
		e.HostChecks = append(e.HostChecks, check)
		writeFullFailureOutput(os.Stdout, profileName, packageName+" host check", out)
		return err
	}
	e.HostChecks = append(e.HostChecks, check)
	return nil
}

func runFullPanicHostCheck(root, reportPath, profileName string, p profile, goRoot, artifact string, e *fullPackage, run func(string, command) ([]byte, error)) error {
	return runFullHostCheck(root, reportPath, profileName, "test/go", "test_go_panic_child.log", "unrecovered init panic traceback", fullPanicCommand(p, root, goRoot, artifact), e, func(out []byte, err error) error {
		return validateFullPanic(root, out, err)
	}, run)
}

func runFullFinalizerHostCheck(root, reportPath, profileName string, p profile, goRoot, artifact, name string, e *fullPackage, run func(string, command) ([]byte, error)) error {
	logName := "test_go_finalizer_invalid_" + strings.ReplaceAll(name, " ", "_") + ".log"
	return runFullHostCheck(root, reportPath, profileName, "test/go", logName, "SetFinalizer rejects "+name, fullFinalizerInvalidCommand(p, root, goRoot, artifact, name), e, func(out []byte, err error) error {
		return validateFullFinalizerInvalid(out, err, name)
	}, run)
}

func runFullBuiltinPrintHostCheck(root, reportPath, profileName string, p profile, goRoot, artifact string, e *fullPackage, run func(string, command) ([]byte, error)) error {
	return runFullHostCheck(root, reportPath, profileName, "test", "test_builtin_print_child.log", "Go 1.26 builtin print formatting", fullBuiltinPrintCommand(p, root, goRoot, artifact), e, validateFullBuiltinPrint, run)
}

func runFullGoexitHostCheck(root, reportPath, profileName string, p profile, goRoot, artifact string, e *fullPackage, run func(string, command) ([]byte, error)) error {
	return runFullHostCheck(root, reportPath, profileName, "test/llgoext", "test_llgoext_goexit_lifecycle_child.log", "main Goexit releases its logical goroutine once", fullGoexitLifecycleCommand(p, root, goRoot, artifact), e, validateFullGoexitLifecycle, run)
}
