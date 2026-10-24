package build

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"golang.org/x/tools/cover"
)

func TestCoverageBuildAgainstGo(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs covered applications")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	const fixture = "./internal/build/testdata/coverageapp"
	const mainPackage = "github.com/xgo-dev/llgo/internal/build/testdata/coverageapp"
	for _, tc := range []struct {
		name     string
		mode     string
		packages string
		lto      lto.Mode
		files    bool
		cgo      bool
	}{
		{name: "set", mode: "set"},
		{name: "count", mode: "count"},
		{name: "atomic", mode: "atomic"},
		{name: "named files", mode: "count", files: true},
		{name: "named cgo files", mode: "atomic", cgo: true},
		// Include main in gc comparisons: Go 1.27 currently omits its exit
		// hook when -coverpkg selects only dependencies. LLGo must still
		// install that hook without counting main; test that separately below.
		{
			name:     "selected dependency",
			mode:     "count",
			packages: mainPackage + ",github.com/xgo-dev/llgo/internal/build/testdata/coverage/dep",
		},
		{
			name:     "external module",
			mode:     "atomic",
			packages: mainPackage + ",golang.org/x/mod/semver",
		},
		{name: "thin LTO", mode: "count", lto: lto.Thin},
		{name: "full LTO", mode: "atomic", lto: lto.Full},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			input := fixture
			if tc.files {
				input += "/main.go"
			}
			if tc.cgo {
				input = filepath.Join(dir, "main.go")
				const source = `package main

// static int add(int a, int b) { return a + b; }
import "C"
import "os"

func main() {
	if C.add(20, 22) != 42 {
		panic("incorrect cgo result")
	}
	if len(os.Args) > 1 && os.Args[1] == "exit" {
		os.Exit(7)
	}
}
`
				if err := os.WriteFile(input, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var profiles []string
			var outputs [][]byte
			for _, compiler := range []string{"go", "llgo"} {
				app := filepath.Join(dir, compiler)
				if runtime.GOOS == "windows" {
					app += ".exe"
				}
				if compiler == "go" {
					args := []string{"build", "-covermode=" + tc.mode, "-o", app}
					if tc.packages != "" {
						args = append(args, "-coverpkg="+tc.packages)
					}
					cmd := exec.Command("go", append(args, input)...)
					cmd.Dir = root
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("go build: %v\n%s", err, output)
					}
				} else {
					conf := coverageTestConfig(t, ModeBuild)
					conf.OutFile = app
					conf.Coverage = &CoverageConfig{Mode: tc.mode, Packages: tc.packages}
					conf.LTO = tc.lto
					if tc.lto != lto.Off {
						conf.OptLevel = optlevel.O2
					}
					output := captureCoverageOutput(t, func() {
						_, err := Build(Invocation{Args: []string{input}, Dir: root, Config: conf})
						if err != nil {
							t.Fatal(err)
						}
					})
					if len(output) != 0 {
						t.Fatalf("build printed a test report: %s", output)
					}
				}
				data := filepath.Join(dir, compiler+"-data")
				if err := os.Mkdir(data, 0700); err != nil {
					t.Fatal(err)
				}
				var allOutput []byte
				for _, args := range [][]string{nil, {"yes"}, {"exit"}} {
					cmd := exec.Command(app, args...)
					cmd.Env = withEnv(os.Environ(), "GOCOVERDIR="+data)
					output, err := cmd.CombinedOutput()
					if len(args) != 0 && args[0] == "exit" {
						var exitErr *exec.ExitError
						if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
							t.Fatalf("%s exit: %v\n%s", compiler, err, output)
						}
					} else if err != nil {
						t.Fatalf("%s run: %v\n%s", compiler, err, output)
					}
					allOutput = append(allOutput, output...)
				}
				outputs = append(outputs, allOutput)
				counters, err := filepath.Glob(filepath.Join(data, "covcounters.*"))
				if err != nil || len(counters) != 3 {
					t.Fatalf("%s: normal return and os.Exit must emit counters: %v, %v", compiler, counters, err)
				}
				profile := filepath.Join(dir, compiler+".out")
				cmd := exec.Command("go", "tool", "covdata", "textfmt", "-i="+data, "-o="+profile)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s covdata: %v\n%s", compiler, err, output)
				}
				profiles = append(profiles, profile)
				if tc.name == "set" {
					checkCoverageBuildExitBehavior(t, app)
				}
			}
			if !bytes.Equal(outputs[0], outputs[1]) {
				t.Fatalf("application output differs:\ngo: %s\nllgo: %s", outputs[0], outputs[1])
			}
			compareCoverageProfiles(t, profiles[0], profiles[1])
		})
	}
}

func checkCoverageBuildExitBehavior(t *testing.T, app string) {
	t.Helper()
	cmd := exec.Command(app)
	cmd.Env = withEnv(os.Environ(), "GOCOVERDIR=")
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "warning: GOCOVERDIR not set, no coverage data emitted") {
		t.Fatalf("missing coverage-directory warning: %v\n%s", err, output)
	}
	data := t.TempDir()
	cmd = exec.Command(app, "panic")
	cmd.Env = withEnv(os.Environ(), "GOCOVERDIR="+data)
	output, err = cmd.CombinedOutput()
	if err == nil || !bytes.Contains(output, []byte("coverage crash fixture")) {
		t.Fatalf("panic fixture: %v\n%s", err, output)
	}
	counters, err := filepath.Glob(filepath.Join(data, "covcounters.*"))
	if err != nil || len(counters) != 0 {
		t.Fatalf("unrecovered panic unexpectedly ran exit hooks: %v, %v", counters, err)
	}
}

func TestCoverageBuildUncoveredMain(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs covered applications")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	const dependency = "github.com/xgo-dev/llgo/internal/build/testdata/coverage/dep"
	conf := coverageTestConfig(t, ModeBuild)
	conf.OutFile = filepath.Join(dir, "app")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	conf.Coverage = &CoverageConfig{Packages: dependency}
	_, err = Build(Invocation{
		Args:   []string{"./internal/build/testdata/coverageapp"},
		Dir:    root,
		Config: conf,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(conf.OutFile)
	cmd.Env = withEnv(os.Environ(), "GOCOVERDIR="+dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	profile := filepath.Join(dir, "cover.out")
	cmd = exec.Command("go", "tool", "covdata", "textfmt", "-i="+dir, "-o="+profile)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("covdata: %v\n%s", err, output)
	}
	profiles, err := cover.ParseProfiles(profile)
	if err != nil || len(profiles) != 1 || profiles[0].FileName != dependency+"/dep.go" {
		t.Fatalf("unselected main must flush only selected dependency: %v, %v", profiles, err)
	}
	var hits int
	for _, block := range profiles[0].Blocks {
		hits += block.Count
	}
	if hits == 0 {
		t.Fatal("dependency initialization was not counted")
	}
}

func TestCoverageBuildAllFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("instruments an application's complete dependency graph")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	conf := coverageTestConfig(t, ModeBuild)
	conf.OutFile = filepath.Join(dir, "app")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	conf.Coverage = &CoverageConfig{Mode: "atomic", Packages: "all"}
	_, err = Build(Invocation{
		Args:   []string{"./internal/build/testdata/coveragefiles/main.go"},
		Dir:    root,
		Config: conf,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(conf.OutFile)
	cmd.Env = withEnv(os.Environ(), "GOCOVERDIR="+dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, output)
	}
	cmd = exec.Command("go", "tool", "covdata", "percent", "-i="+dir)
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("command-line-arguments")) ||
		!bytes.Contains(output, []byte("strconv")) {
		t.Fatalf("-coverpkg=all omitted named files: %v\n%s", err, output)
	}
	profile := filepath.Join(dir, "cover.out")
	for _, args := range [][]string{
		{"tool", "covdata", "textfmt", "-i=" + dir, "-o=" + profile},
		{"tool", "cover", "-html=" + profile, "-o=" + filepath.Join(dir, "cover.html")},
	} {
		cmd = exec.Command("go", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("coverage report: %v\n%s", err, output)
		}
	}
}
