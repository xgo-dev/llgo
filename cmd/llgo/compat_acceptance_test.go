//go:build !llgo

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestNativeUnqualifiedTestControlsFailEarly(t *testing.T) {
	dir := t.TempDir()
	for _, control := range []string{"cpu=1", "fuzz=FuzzOne", "fuzztime=1x", "fuzzminimizetime=1x", "fuzzworker=true", "fuzzcachedir=cache", "memprofilerate=2"} {
		for _, args := range [][]string{
			{"test", "-" + control, "./nonexistent-package"},
			{"test", "./nonexistent-package", "-" + control},
			{"test", "-test." + control, "./nonexistent-package"},
			{"test", "./nonexistent-package", "-args", "-test." + control},
		} {
			// fuzzworker/fuzzcachedir are private test plumbing, not public
			// driver aliases; their reserved binary spelling is still rejected.
			if strings.HasPrefix(control, "fuzzworker") || strings.HasPrefix(control, "fuzzcachedir") {
				if strings.HasPrefix(args[1], "-test.") {
					continue
				}
			}
			out, code := nativeCommand(t, dir, nil, args...)
			if code == 0 || !strings.Contains(out, "unsupported") || strings.Contains(out, "nonexistent-package") {
				t.Fatalf("%v: exit=%d output=%s", args, code, out)
			}
		}
		out, code := nativeCommand(t, dir, []string{"GOFLAGS=-" + control}, "test", "./nonexistent-package")
		if code == 0 || !strings.Contains(out, "unsupported") || strings.Contains(out, "nonexistent-package") {
			t.Fatalf("GOFLAGS=%s: exit=%d output=%s", control, code, out)
		}
	}
	for _, args := range [][]string{
		{"install", "example.invalid/tool@v1.0.0"},
		{"test", "-buildmode=c-shared", "./nonexistent-package"},
		{"test", "-target=wasi", "-cpuprofile=cpu.out", "./nonexistent-package"},
		{"test", "-target=wasi", "./nonexistent-package", "-args", "-test.memprofile=mem.out"},
	} {
		out, code := nativeCommand(t, dir, nil, args...)
		if code == 0 || strings.Contains(out, "nonexistent-package") {
			t.Fatalf("early contract check %v: exit=%d output=%s", args, code, out)
		}
	}
}

func TestNativeTestSelectionAndBenchmarkConsumers(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/workflow\n\ngo 1.27.0\n")
	nativeFile(t, filepath.Join(dir, "workflow_test.go"), `package workflow
import ("runtime"; "sync/atomic"; "testing")
func TestProtocol(t *testing.T) {
 if !testing.Short() { t.Fatal("-short was lost") }
 if runtime.MemProfileRate != 1 { t.Fatal("full allocation profiling was lost") }
 var active, peak atomic.Int32
 gate := make(chan struct{})
 for i:=0; i<4; i++ { t.Run("parallel", func(t *testing.T) {
  t.Parallel()
  n:=active.Add(1)
  for old:=peak.Load(); n>old && !peak.CompareAndSwap(old,n); old=peak.Load() {}
  if n==2 { select { case <-gate: default: close(gate) } }
  <-gate
  if n>2 { t.Error("-parallel limit was lost") }
  active.Add(-1)
 }) }
 t.Cleanup(func(){if peak.Load()!=2 { t.Error("parallel tests did not overlap") }})
}
func TestProtocolSkip(t *testing.T) { t.Fatal("-skip was lost") }
func TestOther(t *testing.T) { t.Fatal("-run was lost") }
func TestFailFirst(t *testing.T) { t.Fatal("intentional failure") }
func TestFailSecond(t *testing.T) { t.Fatal("failfast did not stop the next test") }
func TestFullpath(t *testing.T) { t.Error("full path witness") }
func BenchmarkProtocol(b *testing.B) { for i:=0;i<b.N;i++ {} }
func FuzzSeed(f *testing.F) { f.Add(1); f.Fuzz(func(t *testing.T, n int){if n!=1 {t.Fatal(n)}}) }
`)
	out, code := nativeCommand(t, dir, nil, "test", ".", "-v", "-run=^TestProtocol", "-skip=Skip$", "-short", "-count=2", "-shuffle=1", "-parallel=2", "-memprofilerate=1")
	if code != 0 || strings.Count(out, "--- PASS: TestProtocol (") != 2 || !strings.Contains(out, "-test.shuffle 1") {
		t.Fatalf("selection/repetition/parallelism: exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "test", ".", "-list=Protocol|FuzzSeed")
	if code != 0 || !strings.Contains(out, "TestProtocol\n") || !strings.Contains(out, "BenchmarkProtocol\n") || !strings.Contains(out, "FuzzSeed\n") || strings.Contains(out, "=== RUN") {
		t.Fatalf("test listing: exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "test", ".", "-run=^$", "-bench=^BenchmarkProtocol$", "-benchtime=2x", "-benchmem")
	if code != 0 || !regexp.MustCompile(`(?m)^BenchmarkProtocol(?:-\d+)?\s+2\s+`).MatchString(out) || !strings.Contains(out, "allocs/op") {
		t.Fatalf("benchmark controls: exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "test", ".", "-v", "-run=^TestFail", "-failfast")
	if code == 0 || !strings.Contains(out, "intentional failure") || strings.Contains(out, "RUN   TestFailSecond") {
		t.Fatalf("failfast: exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "test", ".", "-run=^TestFullpath$", "-fullpath")
	// Compiler source positions use slash separators on every host.
	if code == 0 || !strings.Contains(out, filepath.ToSlash(filepath.Join(dir, "workflow_test.go"))) {
		t.Fatalf("fullpath: exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "test", ".", "-v", "-run=^FuzzSeed$")
	if code != 0 || !strings.Contains(out, "PASS: FuzzSeed/seed#0") {
		t.Fatalf("seed execution without fuzz engine: exit=%d output=%s", code, out)
	}
}

func TestNativeRunAndInstallVerbosity(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0700); err != nil {
		t.Fatal(err)
	}
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/cliworkflow\n\ngo 1.27.0\n")
	nativeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){println(\"run-install-consumer\")}\n")
	for _, command := range []string{"run", "install"} {
		out, code := nativeCommand(t, dir, []string{"GOBIN=" + binDir}, command, "-v", "-a", "-p=1", ".")
		if code != 0 || !strings.Contains(out, "example.com/cliworkflow") || strings.Contains(out, "==> BuildSSA") {
			t.Fatalf("%s package verbosity: exit=%d output=%s", command, code, out)
		}
		if command == "run" && !strings.Contains(out, "run-install-consumer") {
			t.Fatalf("run output was lost: %s", out)
		}
	}
	program := filepath.Join(binDir, "cliworkflow")
	if runtime.GOOS == "windows" {
		program += ".exe"
	}
	if out, err := exec.Command(program).CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "run-install-consumer" {
		t.Fatalf("installed native artifact: %v, %s", err, out)
	}
}

func TestNativeTestLibraryArtifactsRequireCompileOnly(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/testlibrary\n\ngo 1.27.0\n")
	nativeFile(t, filepath.Join(dir, "library_test.go"), "package library\nimport \"testing\"\nfunc TestMustNotRun(t *testing.T){t.Fatal(\"compile-only ran a test\")}\n")
	for _, mode := range []string{"c-archive", "c-shared"} {
		output := filepath.Join(dir, mode)
		out, code := nativeCommand(t, dir, nil, "test", "-c", "-O0", "-buildmode="+mode, "-o="+output, ".")
		if code != 0 || strings.Contains(out, "compile-only ran a test") {
			t.Fatalf("%s: exit=%d output=%s", mode, code, out)
		}
		if info, err := os.Stat(output); err != nil || info.Size() == 0 {
			t.Fatalf("missing %s artifact: %v", mode, err)
		}
	}
}

func TestNativeProfilesRetainBinaryAndUseInvocationDirectory(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/profiles\n\ngo 1.27.0\n")
	if err := os.Mkdir(filepath.Join(dir, "pkg"), 0700); err != nil {
		t.Fatal(err)
	}
	nativeFile(t, filepath.Join(dir, "pkg", "profile_test.go"), `package fixture
import ("testing"; "time")
var data []byte
var sink int
func TestProfile(t *testing.T) {
 data=make([]byte,1<<20)
 until:=time.Now().Add(150*time.Millisecond)
 for time.Now().Before(until) { for i:=0;i<1000;i++ {sink+=i} }
}
`)
	for _, output := range []string{"", "retained"} {
		profileDir := filepath.Join(dir, "profiles"+output)
		if err := os.Mkdir(profileDir, 0700); err != nil {
			t.Fatal(err)
		}
		args := []string{"test", "./pkg", "-cpuprofile=cpu.prof", "-memprofile=mem.prof", "-outputdir=" + filepath.Base(profileDir)}
		binary := filepath.Join(dir, "pkg.test")
		if output != "" {
			args = append(args, "-test.memprofilerate=1")
			binary = filepath.Join(dir, output)
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			args = append(args, "-o="+binary)
		} else if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		out, code := nativeCommand(t, dir, nil, args...)
		if code != 0 {
			t.Fatalf("profile lifecycle: exit=%d output=%s", code, out)
		}
		if info, err := os.Stat(binary); err != nil || info.Size() == 0 {
			t.Fatalf("profile binary was not retained: %s: %v", binary, err)
		}
		for _, name := range []string{"cpu.prof", "mem.prof"} {
			path := filepath.Join(profileDir, name)
			cmd := exec.Command("go", "tool", "pprof", "-raw", path)
			data, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("invalid %s: %v\n%s", name, err, data)
			}
			if name == "mem.prof" && !bytes.Contains(data, []byte("Period: 1\n")) {
				t.Fatalf("full allocation counts were rescaled: %s", data)
			}
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "second"), 0700); err != nil {
		t.Fatal(err)
	}
	nativeFile(t, filepath.Join(dir, "second", "value.go"), "package second\n")
	out, code := nativeCommand(t, dir, nil, "test", "./...", "-memprofile=multiple.prof")
	if code == 0 || !strings.Contains(out, "multiple test packages") {
		t.Fatalf("multi-package profile: exit=%d output=%s", code, out)
	}
}

func TestDelegatedCommandStreamsAndExitStatus(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/delegation\n\ngo 1.27.0\n")
	nativeFile(t, filepath.Join(dir, "main.go"), "package main\n//go:generate go env GOHOSTOS\nfunc main() {}\n")
	nativeFile(t, filepath.Join(dir, "go.work"), "go 1.27.0\nuse .\n")
	cases := [][]string{{"env", "GOHOSTOS"}, {"fix", "-diff", "."}, {"get", "."}, {"vet", "."}, {"fmt", "-n", "."}, {"doc", "fmt.Println"}, {"generate", "-n", "."}, {"list", "-f={{.ImportPath}}", "."}, {"mod", "edit", "-json"}, {"work", "edit", "-json", "go.work"}, {"version", "missing.exe"}}
	for _, command := range []string{"env", "fmt", "doc", "fix", "generate", "get", "list", "vet", "mod", "work", "version"} {
		cases = append(cases, []string{command, "-h"}, []string{command, "-unknown-llgo-contract-flag"})
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var goOut, goErr, llOut, llErr bytes.Buffer
			want := exec.Command("go", args...)
			want.Dir = dir
			want.Env = append(os.Environ(), "GOENV=off", "GOFLAGS= ", "GOWORK=off", "GOTOOLCHAIN=local")
			want.Stdout, want.Stderr = &goOut, &goErr
			wantErr := want.Run()
			encoded, _ := json.Marshal(args)
			got := exec.Command(self, "-test.run=^TestNativeCommandProcess$")
			got.Dir = dir
			got.Env = append(want.Env, "GOCOVERDIR="+nativeCoverageDir(t), "LLGO_ROOT="+root, "LLGO_TEST_NATIVE_ARGS="+string(encoded))
			got.Stdout, got.Stderr = &llOut, &llErr
			gotErr := got.Run()
			code := func(err error) int {
				if err == nil {
					return 0
				}
				if e, ok := err.(*exec.ExitError); ok {
					return e.ExitCode()
				}
				t.Fatalf("command setup: %v", err)
				return -1
			}
			if code(wantErr) != code(gotErr) || goOut.String() != llOut.String() || goErr.String() != llErr.String() {
				t.Fatal(fmt.Sprintf("Go=(%d,stdout=%q,stderr=%q)\nLLGo=(%d,stdout=%q,stderr=%q)", code(wantErr), goOut.String(), goErr.String(), code(gotErr), llOut.String(), llErr.String()))
			}
		})
	}
}
