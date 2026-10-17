//go:build !llgo

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestNativeCommandProcess(t *testing.T) {
	if value := os.Getenv("LLGO_TEST_NATIVE_ARGS"); value != "" {
		var args []string
		if err := json.Unmarshal([]byte(value), &args); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("LLGO_TEST_NATIVE_NO_GO") == "1" {
			if tool, err := exec.LookPath("go"); err == nil {
				t.Fatalf("Go toolchain is available in isolated PATH: %s", tool)
			}
		}
		os.Args = append([]string{"llgo"}, args...)
		main()
		os.Exit(0)
	}
}

func nativeCoverageDir(t *testing.T) string {
	t.Helper()
	// Let go test include subprocess execution in the parent's coverage profile.
	if dir := os.Getenv("GOCOVERDIR"); dir != "" {
		return dir
	}
	return t.TempDir()
}

func nativeCommand(t *testing.T, dir string, env []string, args ...string) (string, int) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(args)
	cmd := exec.Command(self, "-test.run=^TestNativeCommandProcess$")
	cmd.Dir = dir
	root, _ := filepath.Abs("../..")
	coverage := nativeCoverageDir(t)
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverage, "LLGO_ROOT="+root, "GOENV=off", "GOFLAGS= ", "GOWORK=off", "GOTOOLCHAIN=local", "LLGO_TEST_NATIVE_ARGS="+string(data))
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return string(out), code
}

func nativeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNativeCompilerLanguageFlagSpellings(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/gcflaglang\ngo 1.27.0\n")
	nativeFile(t, filepath.Join(dir, "main.go"), `package main
func main() {
    var fs []func() int
    for i := 0; i < 3; i++ {
        fs = append(fs, func() int { return i })
    }
    for _, f := range fs {
        println(f())
    }
}
`)
	out, code := nativeCommand(t, dir, nil, "run", ".")
	if code != 0 || strings.Join(strings.Fields(out), " ") != "0 1 2" {
		t.Fatalf("module language semantics: exit=%d output=%s", code, out)
	}
	for _, value := range []string{"-lang=go1.21", "-lang go1.21"} {
		for _, defaults := range []bool{false, true} {
			source := "CLI"
			if defaults {
				source = "GOFLAGS"
			}
			t.Run(source+"/"+value, func(t *testing.T) {
				args := []string{"run", "-gcflags=" + value, "."}
				var env []string
				if defaults {
					env = []string{"GOFLAGS=" + strconv.Quote(args[1])}
					args = []string{"run", "."}
				}
				out, code := nativeCommand(t, dir, env, args...)
				if code != 0 || strings.Join(strings.Fields(out), " ") != "3 3 3" {
					t.Fatalf("compiler language semantics: exit=%d output=%s", code, out)
				}
			})
		}
	}
}

func TestNativeUnsupportedControlsFailBeforeLoading(t *testing.T) {
	dir := t.TempDir()
	for _, command := range []string{"build", "install", "run", "cmptest", "test"} {
		for _, control := range []string{"-race", "-msan", "-asan", "-pgo=off", "-trimpath", "-work", "-n", "-gcflags=-m", "-ldflags=-X main.Version=1"} {
			t.Run(command+control, func(t *testing.T) {
				out, code := nativeCommand(t, dir, nil, command, control, "./nonexistent-package")
				if code == 0 || !strings.Contains(out, "unsupported") || strings.Contains(out, "nonexistent-package") {
					t.Fatalf("exit=%d output=%s", code, out)
				}
			})
		}
	}
	for _, control := range []string{"-trace=trace.out", "-blockprofile=block.out", "-mutexprofile=mutex.out", "-blockprofilerate=1", "-mutexprofilefraction=1"} {
		for _, args := range [][]string{{"test", control, "./nonexistent-package"}, {"test", "./nonexistent-package", "-args", "-test." + strings.TrimPrefix(control, "-")}} {
			out, code := nativeCommand(t, dir, nil, args...)
			if code == 0 || !strings.Contains(out, "unsupported") {
				t.Fatalf("%v exit=%d output=%s", args, code, out)
			}
		}
	}
	for _, control := range []string{"-vet=off"} {
		out, code := nativeCommand(t, dir, nil, "test", "./nonexistent-package", control)
		if code == 0 || !strings.Contains(out, "unsupported") || strings.Contains(out, "nonexistent-package") {
			t.Fatalf("post-package %s exit=%d output=%s", control, code, out)
		}
	}
	for _, command := range []string{"clean", "monitor"} {
		out, code := nativeCommand(t, dir, nil, command, "-no-such-flag")
		if code != 1 || !strings.Contains(out, "flag provided but not defined") {
			t.Fatalf("%s exit=%d output=%s", command, code, out)
		}
	}
	out, code := nativeCommand(t, dir, nil, "clean", "-x")
	if code != 1 || !strings.Contains(out, "unsupported") {
		t.Fatalf("clean -x exit=%d output=%s", code, out)
	}
	for _, control := range []string{"-D=tmp", "-I=tmp", "-o=out.o", "-p=main", "-importcfg=imports"} {
		out, code := nativeCommand(t, dir, nil, "tool", "compile", control, "missing.go")
		if code != 2 || !strings.Contains(out, "unsupported") {
			t.Fatalf("%s exit=%d output=%s", control, code, out)
		}
	}
	out, code = nativeCommand(t, dir, nil, "tool", "unknown")
	if code == 0 || !strings.Contains(out, "unknown") {
		t.Fatalf("tool exit=%d output=%s", code, out)
	}
}

func TestNativeVersionInspectionDiagnostics(t *testing.T) {
	dir := t.TempDir()
	want := exec.Command("go", "version", "missing.exe")
	want.Dir = dir
	wantOutput, err := want.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("Go missing-file fixture: %v, %s", err, wantOutput)
	}
	out, code := nativeCommand(t, dir, nil, "version", "missing.exe")
	if code != exit.ExitCode() || out != string(wantOutput) {
		t.Fatalf("version: llgo=(%d,%q), go=(%d,%q)", code, out, exit.ExitCode(), wantOutput)
	}
}

func TestMonitorHelpDoesNotRequireGoDefaultsOrToolchain(t *testing.T) {
	noGoPath := ""
	if runtime.GOOS == "windows" {
		// MinGW test binaries need runtime DLLs from PATH before main runs.
		// Remove Go tool directories while retaining those DLL search paths.
		var paths []string
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if _, err := exec.LookPath(filepath.Join(dir, "go")); err != nil {
				paths = append(paths, dir)
			}
		}
		noGoPath = strings.Join(paths, string(os.PathListSeparator))
	}
	for _, defaults := range []string{"", "-mod=mod -count=1 -tags=custom"} {
		out, code := nativeCommand(t, t.TempDir(), []string{"GOFLAGS=" + defaults, "GOENV=" + filepath.Join(t.TempDir(), "missing-goenv"), "PATH=" + noGoPath, "LLGO_TEST_NATIVE_NO_GO=1"}, "monitor", "-h")
		if code != 0 || !strings.Contains(out, "Usage") {
			t.Fatalf("monitor help GOFLAGS=%q exit=%d output=%s", defaults, code, out)
		}
	}
}

func TestNativeGOFLAGSCannotBypassUnsupportedControls(t *testing.T) {
	dir := t.TempDir()
	for _, value := range []string{"-race", "-gcflags=-m", "-pgo=off"} {
		for _, command := range []string{"build", "install", "run", "test"} {
			out, code := nativeCommand(t, dir, []string{"GOFLAGS=" + value}, command, "./missing")
			if code == 0 || !strings.Contains(out, "unsupported") {
				t.Fatalf("%s GOFLAGS=%s exit=%d output=%s", command, value, code, out)
			}
		}
	}
	goenv := filepath.Join(dir, "goenv")
	nativeFile(t, goenv, "GOFLAGS=-race\n")
	out, code := nativeCommand(t, dir, []string{"GOENV=" + goenv, "GOFLAGS="}, "build", "./missing")
	if code == 0 || !strings.Contains(out, "unsupported") {
		t.Fatalf("persisted GOFLAGS exit=%d output=%s", code, out)
	}
}

func TestNativeDirectoryAndOverlaySourceSelection(t *testing.T) {
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	nativeFile(t, filepath.Join(project, "go.mod"), "module example.com/overlay\ngo 1.24\n")
	nativeFile(t, filepath.Join(project, "main.go"), "package main\nimport \"fmt\"\nfunc main(){fmt.Println(Value+Added)}\n")
	nativeFile(t, filepath.Join(project, "value.go"), "package main\nconst Value=1\n")
	nativeFile(t, filepath.Join(project, "removed.go"), "package main\nthis source must be deleted\n")
	nativeFile(t, filepath.Join(project, "replacement.txt"), "package main\nconst Value=40\n")
	nativeFile(t, filepath.Join(project, "addition.txt"), "package main\nconst Added=2\n")
	nativeFile(t, filepath.Join(project, "overlay.json"), `{"Replace":{"value.go":"replacement.txt","added.go":"addition.txt","removed.go":""}}`)
	artifact := "result"
	if runtime.GOOS == "windows" {
		artifact += ".exe"
	}
	for _, command := range []string{"build", "run"} {
		args := []string{command, "-C", project, "-overlay=overlay.json"}
		if command == "build" {
			args = append(args, "-o="+artifact)
		}
		args = append(args, ".")
		env := []string(nil)
		if command == "build" {
			env = []string{"GOFLAGS=-run=Selected"}
		}
		out, code := nativeCommand(t, dir, env, args...)
		if code != 0 {
			t.Fatalf("%s exit=%d output=%s", command, code, out)
		}
		if command == "build" {
			output, err := exec.Command(filepath.Join(project, artifact)).CombinedOutput()
			if err != nil {
				t.Fatalf("run overlay artifact: %v %s", err, output)
			}
			out = string(output)
		}
		if strings.TrimSpace(out) != "42" {
			t.Fatalf("%s used original source: %s", command, out)
		}
	}
	out, code := nativeCommand(t, dir, []string{"GOFLAGS=-overlay=overlay.json"}, "run", "-C="+project, ".")
	if code != 0 || strings.TrimSpace(out) != "42" {
		t.Fatalf("GOFLAGS overlay exit=%d output=%s", code, out)
	}
	out, code = nativeCommand(t, dir, nil, "run", "-C", project, "-overlay=missing.json", ".")
	if code == 0 || !strings.Contains(out, "read -overlay") {
		t.Fatalf("missing overlay exit=%d output=%s", code, out)
	}
}

func TestNativeModCacheWritable(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for path, data := range map[string]string{"go.mod": "module example.com/cachedep\ngo 1.24\n", "value.go": "package cachedep\nconst Value=42\n"} {
		writer, err := zw.Create("example.com/cachedep@v1.0.0/" + path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/example.com/cachedep/@v/v1.0.0.mod":
			fmt.Fprint(w, "module example.com/cachedep\ngo 1.24\n")
		case "/example.com/cachedep/@v/v1.0.0.info":
			fmt.Fprint(w, `{"Version":"v1.0.0","Time":"2026-01-01T00:00:00Z"}`)
		case "/example.com/cachedep/@v/v1.0.0.zip":
			w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer proxy.Close()
	for _, writable := range []bool{false, true} {
		t.Run(fmt.Sprint(writable), func(t *testing.T) {
			dir := t.TempDir()
			cache := filepath.Join(dir, "cache")
			t.Cleanup(func() {
				_ = filepath.WalkDir(cache, func(path string, entry os.DirEntry, err error) error {
					if err == nil && entry.IsDir() {
						_ = os.Chmod(path, 0755)
					}
					return nil
				})
			})
			nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/cachemain\ngo 1.24\nrequire example.com/cachedep v1.0.0\n")
			nativeFile(t, filepath.Join(dir, "main.go"), "package main\nimport (\"fmt\";\"example.com/cachedep\")\nfunc main(){fmt.Println(cachedep.Value)}\n")
			args := []string{"run", "-mod=mod"}
			if writable {
				args = append(args, "-modcacherw")
			}
			args = append(args, ".")
			out, code := nativeCommand(t, dir, []string{"GOMODCACHE=" + cache, "GOPROXY=" + proxy.URL, "GOSUMDB=off"}, args...)
			if code != 0 || !strings.HasSuffix(strings.TrimSpace(out), "42") {
				t.Fatalf("exit=%d output=%s", code, out)
			}
			info, err := os.Stat(filepath.Join(cache, "example.com/cachedep@v1.0.0"))
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm()&0200 != 0; runtime.GOOS != "windows" && got != writable {
				t.Fatalf("module writable=%v, want %v", got, writable)
			}
		})
	}
}

func TestDocDelegationParity(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"-short", "fmt.Println"}, {"-h"}, {"-unknown"}, {"no-such-package.invalid/symbol"}} {
		cmd := exec.Command("go", append([]string{"doc"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS= ", "GOWORK=off", "GOTOOLCHAIN=local")
		output, err := cmd.CombinedOutput()
		wantCode := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				wantCode = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		out, code := nativeCommand(t, dir, nil, append([]string{"doc"}, args...)...)
		if code != wantCode || out != string(output) {
			t.Fatalf("doc %v: llgo=(%d,%q), go=(%d,%q)", args, code, out, wantCode, output)
		}
	}
	doc := &Cmd_doc{App: new(App)}
	doc.Main("doc")
	if !doc.DisableFlagParsing || doc.Run == nil {
		t.Fatal("doc does not delegate parsing")
	}
}

func TestCleanRejectsInvalidPackages(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/clean\ngo 1.24\n")
	out, code := nativeCommand(t, dir, nil, "clean", "-C", dir, "./missing")
	if code != 1 || !strings.Contains(out, "missing") {
		t.Fatalf("clean exit=%d output=%s", code, out)
	}
}

func TestNativeGOFLAGSDriverDefaultsStayOutOfLoader(t *testing.T) {
	dir := t.TempDir()
	nativeFile(t, filepath.Join(dir, "go.mod"), "module example.com/testdefaults\ngo 1.24\n")
	nativeFile(t, filepath.Join(dir, "value_test.go"), `package testdefaults
import ("os";"testing")
func TestSelected(t *testing.T){if os.Getenv("GOFLAGS")!="-run=Selected -count=1"{t.Fatal("changed program environment")}}
func TestOther(t *testing.T){t.Fatal("GOFLAGS test selection was lost")}
`)
	out, code := nativeCommand(t, dir, []string{"GOFLAGS=-run=Selected -count=1"}, "test", "-v", ".")
	if code != 0 || !strings.Contains(out, "PASS: TestSelected") || strings.Contains(out, "RUN   TestOther") {
		t.Fatalf("test defaults leaked into loader or runner: exit=%d output=%s", code, out)
	}
}
