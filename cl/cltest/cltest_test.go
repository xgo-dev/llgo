package cltest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/littest"
)

func TestNormalizeGoldenNewlines(t *testing.T) {
	got := normalizeGoldenNewlines([]byte("first\r\nsecond\n"))
	want := []byte("first\nsecond\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("normalizeGoldenNewlines() = %q, want %q", got, want)
	}
}

func TestIROnlyGoldenAllowsFinalNewline(t *testing.T) {
	for _, data := range [][]byte{[]byte("; ir-only"), []byte("; ir-only\n"), []byte("; ir-only\r\n")} {
		if !isIROnlyGolden(data) {
			t.Fatalf("isIROnlyGolden(%q) = false", data)
		}
	}
	for _, data := range [][]byte{nil, []byte("\n"), []byte(";"), []byte("; ir-only\n\n")} {
		if isIROnlyGolden(data) {
			t.Fatalf("isIROnlyGolden(%q) = true", data)
		}
	}
}

func TestReadGoldenExecutionModes(t *testing.T) {
	for _, tc := range []struct {
		data string
		run  bool
		diff bool
	}{
		{";", true, false},
		{";\n", true, false},
		{";\r\n", true, false},
		{"; ir-only", false, false},
		{"; ir-only\r\n", false, false},
		{"", true, true},
		{"output\n", true, true},
		{";\n\n", true, true},
		{" ;", true, true},
	} {
		t.Run(fmt.Sprintf("%q", tc.data), func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "expect.txt")
			for _, versioned := range []bool{false, true} {
				path := file
				if versioned {
					path, _ = goldenForGoVersion(file, runtime.Version())
					fallback := "; ir-only"
					if !tc.diff {
						fallback = "literal output\n"
					}
					if err := os.WriteFile(file, []byte(fallback), 0644); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(path, []byte(tc.data), 0644); err != nil {
					t.Fatal(err)
				}
				data, run, err := readGolden(file)
				if err != nil || string(data) != tc.data || run != tc.run || (run && !isRunOnlyGolden(data)) != tc.diff {
					t.Fatalf("readGolden(%q) = %q, %v, %v", path, data, run, err)
				}
			}
		})
	}
}

func TestRuntimeGoldenExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs fixture programs")
	}
	const helperEnv = "LLGO_CLTEST_EXECUTION_DIR"
	if dir := os.Getenv(helperEnv); dir != "" {
		testRunAndTestFrom(t, dir, dir, "", runOptions{checkIR: true, checkOutput: true})
		return
	}
	for _, tc := range []struct {
		name, golden, statement, check, failure string
		runs                                    bool
	}{
		{name: "run only", golden: ";", statement: `println(new(int))`, runs: true},
		{name: "compare", golden: "done\n", statement: `println("done")`, runs: true},
		{name: "output mismatch", golden: "wrong\n", statement: `println("done")`, failure: "unexpected output", runs: true},
		{name: "runtime failure", golden: ";", statement: `panic("run-only panic")`, failure: "run-only panic", runs: true},
		{name: "build failure", golden: ";", statement: `undefinedFunction()`, failure: "undefined: undefinedFunction"},
		{name: "IR only", golden: "; ir-only", statement: `panic("must not run")`},
		{name: "IR mismatch", golden: "; ir-only", statement: `panic("must not run")`, check: "MISSING_IR_INSTRUCTION", failure: "MISSING_IR_INSTRUCTION"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			check := tc.check
			if check == "" {
				check = "define void @main.main("
			}
			source := fmt.Sprintf(`// LITTEST
// CHECK: %s
package main
import "os"
func main() {
	if err := os.WriteFile("executed", []byte("yes"), 0600); err != nil { panic(err) }
	%s
}

`, check, tc.statement)
			for name, data := range map[string]string{"go.mod": "module example.com/execution\ngo 1.20\n", "main.go": source, "expect.txt": tc.golden} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRuntimeGoldenExecution$")
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), helperEnv+"="+dir)
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil || (err != nil) != (tc.failure != "") || !bytes.Contains(output, []byte(tc.failure)) {
				t.Fatalf("helper: %v\n%s", err, output)
			}
			_, err = os.Stat(filepath.Join(dir, "executed"))
			if tc.runs && err != nil || !tc.runs && !os.IsNotExist(err) {
				t.Fatalf("execution marker: %v; want execution=%v", err, tc.runs)
			}
		})
	}
}

func TestEmulatorRunOnlyOutput(t *testing.T) {
	cases := []struct {
		output, golden    string
		emulator, failure bool
	}{
		{"address 0x1234\n", ";", true, false},
		{"boot log\nfatal error\n", ";", true, true},
		{"entry 0x1234\nfatal error: trap\n", ";", true, true},
		{"panic: runtime trap\n", ";", true, true},
		{"handled panic: example\n", ";", true, false},
		{"panic: printed text\n", ";", false, false},
	}
	const helperEnv = "LLGO_CLTEST_EMULATOR_OUTPUT"
	if index := os.Getenv(helperEnv); index != "" {
		for i, tc := range cases {
			if index != fmt.Sprint(i) {
				continue
			}
			conf := build.NewDefaultConf(build.ModeRun)
			conf.Emulator = tc.emulator
			assertExpectedOutput(t, t.TempDir(), []byte(tc.golden), []byte(tc.output), runOptions{conf: conf, filter: func(string) string { return "" }})
			return
		}
		t.Fatal("unknown helper case")
	}
	for i, tc := range cases {
		cmd := exec.Command(os.Args[0], "-test.run=^TestEmulatorRunOnlyOutput$")
		cmd.Env = append(os.Environ(), helperEnv+"="+fmt.Sprint(i))
		output, err := cmd.CombinedOutput()
		if (err != nil) != tc.failure || tc.failure && !bytes.Contains(output, []byte("emulator runtime failure")) {
			t.Fatalf("case %d: %v\n%s", i, err, output)
		}
	}
}

func TestFilterRunOutputToolchainWarnings(t *testing.T) {
	in := []byte("ld64.lld: warning: library is newer than target minimum\n" +
		"'+zcm' is not a recognized feature for this target (ignoring feature)\n" +
		"'+zcz' is not a recognized feature for this target (ignoring feature)\n" +
		"pass\n")
	want := []byte("pass\n")
	if got := filterRunOutput(in); !bytes.Equal(got, want) {
		t.Fatalf("filterRunOutput() = %q, want %q", got, want)
	}
}

func TestAdditionalIRTargets(t *testing.T) {
	targets := []littest.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
	}
	tests := []struct {
		name          string
		currentPrefix string
		want          []littest.Target
	}{
		{name: "listed current", currentPrefix: "DARWIN-ARM64", want: targets[1:]},
		{name: "unlisted current", currentPrefix: "LINUX-ARM64", want: targets},
		{name: "named current", currentPrefix: "TARGET-WASM", want: targets},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := additionalIRTargets(targets, test.currentPrefix); !slices.Equal(got, test.want) {
				t.Fatalf("additionalIRTargets() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadGoldenUsesToolchainVersion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "expect.txt")
	if err := os.WriteFile(file, []byte("default\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	versioned, ok := goldenForGoVersion(file, runtime.Version())
	if !ok {
		t.Fatalf("runtime version %q is not a valid Go version", runtime.Version())
	}
	if err := os.WriteFile(versioned, []byte("versioned\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, check, err := readGolden(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "versioned\n" || !check {
		t.Fatalf("readGolden() = %q, %v, want versioned golden", got, check)
	}

	if err := os.Remove(versioned); err != nil {
		t.Fatal(err)
	}
	got, check, err = readGolden(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "default\n" || !check {
		t.Fatalf("readGolden() fallback = %q, %v, want default golden", got, check)
	}
}

func TestGoldenForGoVersionRejectsUnversionedToolchain(t *testing.T) {
	if got, ok := goldenForGoVersion("expect.txt", "devel custom"); ok || got != "" {
		t.Fatalf("goldenForGoVersion() = %q, %v, want no versioned golden", got, ok)
	}
}

func TestReadGoldenReturnsVersionedReadError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "expect.txt")
	versioned, ok := goldenForGoVersion(file, runtime.Version())
	if !ok {
		t.Fatalf("runtime version %q is not a valid Go version", runtime.Version())
	}
	if err := os.Mkdir(versioned, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := readGolden(file); err == nil {
		t.Fatal("readGolden() succeeded for a versioned golden directory")
	}
}

func TestCaptureDWARFMode(t *testing.T) {
	for _, mode := range []build.DWARFMode{build.DWARFDefault, build.DWARFPreserve, build.DWARFOmit} {
		conf := build.NewDefaultConf(build.ModeRun)
		conf.LinkOptions.DWARF = mode
		got, _, _, _ := withModuleCapture(conf, t.TempDir())
		want := mode
		if want == build.DWARFDefault {
			want = build.DWARFOmit
		}
		if got.LinkOptions.DWARF != want {
			t.Fatalf("capture DWARF = %v, want %v", got.LinkOptions.DWARF, want)
		}
		if conf.LinkOptions.DWARF != mode {
			t.Fatal("capture changed caller options")
		}
	}
}
