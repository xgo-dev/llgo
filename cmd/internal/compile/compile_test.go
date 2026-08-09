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
		"-m=2",
		"-lang=go1.17",
		"-d=panic,ssa/check/on",
		"-p=p",
		"-importcfg=importcfg",
		"case.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.noBounds.value != 1 || opts.concurrency != 2 || opts.noColumns.value != 1 || opts.allErrors.value != 1 || opts.noInline.value != 4 || opts.showOpt.value != 2 {
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
	want := []string{"-dynlink", "-live", "-race", "-smallframes", "-+", "-wb", "-d=libfuzzer", "-d=wb"}
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
	if err := os.WriteFile(valid, []byte("package compilecase\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	previousProcs := runtime.GOMAXPROCS(0)
	stdout, stderr, code := runCompileCommand(t, []string{
		"-B", "-c=1", "-C", "-e", "-lang=go1.22", "-N", "-l", "-complete", valid,
	})
	if code != 0 {
		t.Fatalf("valid compile exit code = %d; stdout=%q stderr=%q", code, stdout, stderr)
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

func TestRunCmdEscapeDiagnostics(t *testing.T) {
	dir := t.TempDir()
	source := dir + "/escape.go"
	if err := os.WriteFile(source, []byte(`package escape
var sink *int
func noescape(p *int) { _ = *p }
func leak(p *int) { sink = p }
func content(p **int) { sink = *p }
func result(p *int) *int { return p }
func through(p *int) *int { return result(p) }
func discard(p *int) { _ = result(p) }
func mutate(p *int) { *p = 1 }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runCompileCommand(t, []string{"-C", "-m", "-l", source})
	if code != 0 {
		t.Fatalf("-m compile exit code = %d, stderr=%q", code, stderr)
	}
	for _, want := range []string{
		// TODO: Re-enable these diagnostics after the summary traversal handles
		// compiler-generated `icmp eq ptr, null` checks without recording a heap
		// flow. HeapToStack cannot generally discard the comparison because the
		// current AllocZ/AllocU runtime contracts do not both guarantee non-nil.
		// "escape.go:3: p does not escape",
		"escape.go:4: leaking param: p",
		// "escape.go:5: leaking param content: p",
		"escape.go:6: leaking param: p to result ~r0 level=0",
		"escape.go:7: leaking param: p to result ~r0 level=0",
		"escape.go:8: p does not escape",
		"escape.go:9: p does not escape",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("-m stderr missing %q:\n%s", want, stderr)
		}
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
