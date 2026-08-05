package compile

import (
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/mockable"
)

func TestGoCompilerFlagNamesAndTypes(t *testing.T) {
	opts := new(options)
	fs := newFlagSet(opts)
	err := fs.Parse([]string{
		"-B",
		"-c=2",
		"-C",
		"-e",
		"-l=4",
		"-lang=go1.17",
		"-d=panic,ssa/check/on",
		"-p=p",
		"-importcfg=importcfg",
		"case.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.noBounds.value != 1 || opts.concurrency != 2 || opts.noColumns.value != 1 || opts.allErrors.value != 1 || opts.noInline.value != 4 {
		t.Fatalf("parsed flags: %+v", opts)
	}
	if opts.lang != "go1.17" || opts.pkgPath != "p" || opts.importCfg != "importcfg" {
		t.Fatalf("parsed flags: %+v", opts)
	}
	if !reflect.DeepEqual(fs.Args(), []string{"case.go"}) {
		t.Fatalf("files=%v", fs.Args())
	}
	if unsupported := opts.unsupported(); len(unsupported) != 0 {
		t.Fatalf("unsupported=%v, want none", unsupported)
	}
}

func TestGoCompilerSpecificFlagIsExplicitlyUnsupported(t *testing.T) {
	opts := new(options)
	fs := newFlagSet(opts)
	if err := fs.Parse([]string{"-d=libfuzzer", "case.go"}); err != nil {
		t.Fatal(err)
	}
	if got := opts.unsupported(); !reflect.DeepEqual(got, []string{"-d=libfuzzer"}) {
		t.Fatalf("unsupported=%v", got)
	}
}

func TestSSACheckSeedDebugSettingsAreCompatible(t *testing.T) {
	opts := &options{debug: stringListFlag{
		"ssa/check/seed,ssa/check/seed=1",
		"ssa/check/seeded=1",
	}}
	want := []string{"-d=ssa/check/seeded=1"}
	if got := opts.unsupported(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unsupported=%v, want %v", got, want)
	}
}

func TestTypeAssertDebugValue(t *testing.T) {
	tests := []struct {
		setting string
		want    int
		ok      bool
	}{
		{setting: "typeassert", want: 1, ok: true},
		{setting: "typeassert=0", want: 0, ok: true},
		{setting: "typeassert:2", want: 2, ok: true},
		{setting: "other", want: 1},
		{setting: "other=1"},
		{setting: "typeassert=bad"},
	}
	for _, tt := range tests {
		t.Run(tt.setting, func(t *testing.T) {
			got, ok := typeAssertDebugValue(tt.setting)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("typeAssertDebugValue(%q) = (%d, %v), want (%d, %v)", tt.setting, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCompatibleDebugSetting(t *testing.T) {
	tests := []struct {
		setting string
		want    bool
	}{
		{setting: "panic", want: true},
		{setting: "ssa/check/on", want: true},
		{setting: "ssa/check/seed", want: true},
		{setting: "ssa/check/seed=1", want: true},
		{setting: "typeassert", want: true},
		{setting: "typeassert=0", want: true},
		{setting: "ssa/check/seeded=1"},
		{setting: "typeassert=bad"},
		{setting: "libfuzzer"},
	}
	for _, tt := range tests {
		t.Run(tt.setting, func(t *testing.T) {
			if got := compatibleDebugSetting(tt.setting); got != tt.want {
				t.Fatalf("compatibleDebugSetting(%q) = %v, want %v", tt.setting, got, tt.want)
			}
		})
	}
}

func TestCountAndListFlags(t *testing.T) {
	var count countFlag
	if !count.IsBoolFlag() {
		t.Fatal("countFlag should accept the bool flag form")
	}
	for _, value := range []string{"true", "true", "false", "4"} {
		if err := count.Set(value); err != nil {
			t.Fatalf("Set(%q): %v", value, err)
		}
	}
	if got := count.String(); got != "4" || !count.set {
		t.Fatalf("count flag = %q, set=%v; want 4, true", got, count.set)
	}
	if err := count.Set("invalid"); err == nil {
		t.Fatal("invalid count value was accepted")
	}

	var list stringListFlag
	if err := list.Set("one"); err != nil {
		t.Fatal(err)
	}
	if err := list.Set("two"); err != nil {
		t.Fatal(err)
	}
	if got := list.String(); got != "one,two" {
		t.Fatalf("list flag = %q, want one,two", got)
	}
}

func TestUnsupportedCompilerFlags(t *testing.T) {
	opts := &options{
		dynlink:     true,
		showOpt:     countFlag{value: 1},
		live:        true,
		race:        true,
		smallFrames: true,
		standard:    true,
		runtimePkg:  true,
		writeBar:    countFlag{set: true},
		debug:       stringListFlag{"panic,libfuzzer", "ssa/check/on,ssa/check/seed=1,wb"},
	}
	want := []string{"-dynlink", "-m", "-live", "-race", "-smallframes", "-+", "-wb", "-d=libfuzzer", "-d=wb"}
	if got := opts.unsupported(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unsupported=%v, want %v", got, want)
	}
}

func TestRunCmdValidationAndVersion(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{name: "bad flag", args: []string{"-unknown"}, wantCode: 2, wantOutput: "flag provided but not defined"},
		{name: "no files", wantCode: 2, wantOutput: "no Go source files"},
		{name: "negative concurrency", args: []string{"-c=-1", "case.go"}, wantCode: 2, wantOutput: "-c must be non-negative"},
		{name: "invalid language", args: []string{"-lang=1.22", "case.go"}, wantCode: 2, wantOutput: "invalid value"},
		{name: "version", args: []string{"-V"}, wantOutput: "compile version llgo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runCompileCommand(t, tt.args)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, tt.wantCode, stdout, stderr)
			}
			if output := stdout + stderr; !strings.Contains(output, tt.wantOutput) {
				t.Fatalf("output %q does not contain %q", output, tt.wantOutput)
			}
		})
	}
}

func TestRunCmdBuildsAndReportsErrors(t *testing.T) {
	dir := t.TempDir()
	valid := dir + "/valid.go"
	if err := os.WriteFile(valid, []byte("package compilecase\nfunc F(v any) int { return v.(int) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	previousProcs := runtime.GOMAXPROCS(0)
	stdout, stderr, code := runCompileCommand(t, []string{
		"-B", "-c=1", "-C", "-d=panic,typeassert", "-e", "-lang=go1.22", "-N", "-l", "-complete", valid,
	})
	if code != 0 {
		t.Fatalf("valid compile exit code = %d; stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, valid+":2: type assertion inlined") {
		t.Fatalf("valid compile stderr = %q; want type assertion diagnostic", stderr)
	}
	if got := runtime.GOMAXPROCS(0); got != previousProcs {
		t.Fatalf("GOMAXPROCS = %d after compile, want restored value %d", got, previousProcs)
	}

	zeroArray := dir + "/zero_array.go"
	if err := os.WriteFile(zeroArray, []byte("package compilecase\ntype A [0]byte\nfunc Get(a *A, i int) byte { return a[i] }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code = runCompileCommand(t, []string{"-B", zeroArray})
	if code != 0 {
		t.Fatalf("-B zero-length array compile exit code = %d; stdout=%q stderr=%q", code, stdout, stderr)
	}

	invalid := dir + "/invalid.go"
	if err := os.WriteFile(invalid, []byte("package compilecase\nvar _ = missing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code = runCompileCommand(t, []string{invalid})
	if code != 1 || !strings.Contains(stderr, "missing") {
		t.Fatalf("invalid compile exit code = %d, stderr=%q; want code 1 and diagnostic", code, stderr)
	}

	invalidPragma := dir + "/invalid_pragma.go"
	if err := os.WriteFile(invalidPragma, []byte("package compilecase\n//go:uintptrkeepalive\nfunc F(uintptr) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code = runCompileCommand(t, []string{"-C", "-std", invalidPragma})
	if code != 1 || !strings.Contains(stderr, "go:uintptrkeepalive requires go:nosplit") {
		t.Fatalf("-std compile exit code = %d, stderr=%q; want code 1 and pragma diagnostic", code, stderr)
	}
}

func runCompileCommand(t *testing.T, args []string) (stdout, stderr string, exitCode int) {
	t.Helper()
	outFile, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	errFile, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outFile, errFile
	mockable.EnableMock()
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
		runCmd(Cmd, args)
	}()
	mockable.DisableMock()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	if exited {
		exitCode = mockable.ExitCode()
	}
	if err := outFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := errFile.Close(); err != nil {
		t.Fatal(err)
	}
	outData, err := os.ReadFile(outFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	errData, err := os.ReadFile(errFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(outData), string(errData), exitCode
}
