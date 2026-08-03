package goroot

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseDirective(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "case.go")
	if err := os.WriteFile(file, []byte("// run\n\npackage main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	directive, args, ok := parseDirective(file)
	if !ok {
		t.Fatal("expected directive to be found")
	}
	if directive != "run" {
		t.Fatalf("directive=%q, want run", directive)
	}
	if len(args) != 0 {
		t.Fatalf("args=%v, want none", args)
	}
}

func TestParseDirectiveWithArgs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "case.go")
	if err := os.WriteFile(file, []byte("// errorcheckandrundir -1\n\npackage ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}
	directive, args, ok := parseDirective(file)
	if !ok {
		t.Fatal("expected directive to be found")
	}
	if directive != "errorcheckandrundir" {
		t.Fatalf("directive=%q, want errorcheckandrundir", directive)
	}
	if !reflect.DeepEqual(args, []string{"-1"}) {
		t.Fatalf("args=%v, want [-1]", args)
	}
}

func TestParseDirectiveQuotedArgs(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "case.go")
	src := `// runindir -gomodversion "1.23" -gcflags='all=-N -l'

package ignored
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	directive, args, ok := parseDirective(file)
	if !ok {
		t.Fatal("expected directive to be found")
	}
	if directive != "runindir" {
		t.Fatalf("directive=%q, want runindir", directive)
	}
	want := []string{"-gomodversion", "1.23", "-gcflags=all=-N -l"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%v, want %v", args, want)
	}
}

func TestReleaseTagsFor(t *testing.T) {
	got := releaseTagsFor("go1.24.11")
	want := []string{
		"go1.1", "go1.2", "go1.3", "go1.4", "go1.5", "go1.6", "go1.7", "go1.8",
		"go1.9", "go1.10", "go1.11", "go1.12", "go1.13", "go1.14", "go1.15", "go1.16",
		"go1.17", "go1.18", "go1.19", "go1.20", "go1.21", "go1.22", "go1.23", "go1.24",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("releaseTagsFor()=%v, want %v", got, want)
	}
}

func TestXFailMatch(t *testing.T) {
	cfg := xfailConfig{
		Entries: []xfailEntry{{
			Version:   "go1.24",
			Platform:  "darwin/arm64",
			Directive: "run",
			Case:      "fixedbugs/*",
			Reason:    "known issue",
		}},
	}
	tc := testCase{RelPath: "fixedbugs/bug123.go", Directive: "run"}
	match, reason := cfg.Match("go1.24.11", "darwin/arm64", tc)
	if !match {
		t.Fatal("expected xfail match")
	}
	if reason != "known issue" {
		t.Fatalf("reason=%q, want known issue", reason)
	}
}

func TestFlakyMatch(t *testing.T) {
	cfg := xfailConfig{
		Flakes: []xfailEntry{{
			Version:   "go1.25",
			Platform:  "linux/amd64",
			Directive: "run",
			Case:      "fixedbugs/issue11256.go",
			Reason:    "flaky issue",
		}},
	}
	tc := testCase{RelPath: "fixedbugs/issue11256.go", Directive: "run"}
	match, reason := cfg.MatchFlaky("go1.25.0", "linux/amd64", tc)
	if !match {
		t.Fatal("expected flaky match")
	}
	if reason != "flaky issue" {
		t.Fatalf("reason=%q, want flaky issue", reason)
	}
}

func TestMatchGoVersion(t *testing.T) {
	tests := []struct {
		version   string
		goVersion string
		want      bool
	}{
		{version: "go1.24", goVersion: "go1.24", want: true},
		{version: "go1.24", goVersion: "go1.24.11", want: true},
		{version: "go1.24", goVersion: "go1.24rc1", want: true},
		{version: "go1.24", goVersion: "go1.24beta1", want: true},
		{version: "go1.2", goVersion: "go1.24.11", want: false},
		{version: "go1.25", goVersion: "go1.24.11", want: false},
	}
	for _, tt := range tests {
		if got := matchGoVersion(tt.version, tt.goVersion); got != tt.want {
			t.Fatalf("matchGoVersion(%q, %q)=%v, want %v", tt.version, tt.goVersion, got, tt.want)
		}
	}
}

func TestHostSkipMatch(t *testing.T) {
	cfg := xfailConfig{
		HostSkips: []xfailEntry{{
			Version:   "go1.24",
			Platform:  "darwin/arm64",
			Directive: "run",
			Case:      "fixedbugs/*",
			Reason:    "host-unsafe",
		}},
	}
	tc := testCase{RelPath: "fixedbugs/bug123.go", Directive: "run"}
	match, reason := cfg.MatchHostSkip("go1.24.11", "darwin/arm64", tc)
	if !match {
		t.Fatal("expected host skip match")
	}
	if reason != "host-unsafe" {
		t.Fatalf("reason=%q, want host-unsafe", reason)
	}
}

func TestTimeoutMatch(t *testing.T) {
	cfg := xfailConfig{
		Timeouts: []timeoutEntry{{
			Version:   "go1.24",
			Platform:  "darwin/arm64",
			Directive: "run",
			Case:      "fixedbugs/*",
			Timeout:   "90s",
			Reason:    "slow case",
		}},
	}
	tc := testCase{RelPath: "fixedbugs/bug123.go", Directive: "run"}
	timeout, reason, ok := cfg.MatchTimeout("go1.24.11", "darwin/arm64", tc)
	if !ok {
		t.Fatal("expected timeout match")
	}
	if timeout != 90*time.Second {
		t.Fatalf("timeout=%s, want 90s", timeout)
	}
	if reason != "slow case" {
		t.Fatalf("reason=%q, want slow case", reason)
	}
}

func TestEffectiveBuildTimeout(t *testing.T) {
	if got := effectiveBuildTimeout(3*time.Minute, 20*time.Second); got != 3*time.Minute {
		t.Fatalf("effectiveBuildTimeout()=%s, want 3m", got)
	}
	if got := effectiveBuildTimeout(3*time.Minute, 4*time.Minute); got != 4*time.Minute {
		t.Fatalf("effectiveBuildTimeout()=%s, want 4m", got)
	}
}

func TestRunProgramTimeout(t *testing.T) {
	if os.Getenv("LLGO_GOROOT_HELPER") == "sleep" {
		time.Sleep(200 * time.Millisecond)
		return
	}
	disableSystemMemoryLimits(t)

	stdout, stderr, exitCode, elapsed, err := runProgram(
		t.TempDir(),
		os.Args[0],
		append(os.Environ(), "LLGO_GOROOT_HELPER=sleep"),
		50*time.Millisecond,
		"-test.run=TestRunProgramTimeout",
	)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if exitCode == 0 {
		t.Fatalf("exitCode=%d, want non-zero on timeout", exitCode)
	}
	if len(stdout) != 0 {
		t.Fatalf("stdout=%q, want empty", stdout)
	}
	if len(stderr) != 0 {
		t.Fatalf("stderr=%q, want empty", stderr)
	}
	if !strings.Contains(err.Error(), "timed out after") {
		t.Fatalf("err=%v, want timeout", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("elapsed=%s, want >= 50ms", elapsed)
	}
}

func TestRunProgramRSSLimit(t *testing.T) {
	if !resourceMonitoringSupported() {
		t.Skip("process-group RSS monitoring is unavailable")
	}
	if os.Getenv("LLGO_GOROOT_HELPER") == "allocate" {
		memory := make([]byte, 64<<20)
		for i := 0; i < len(memory); i += 4096 {
			memory[i] = 1
		}
		time.Sleep(5 * time.Second)
		runtime.KeepAlive(memory)
		return
	}
	disableSystemMemoryLimits(t)

	oldLimit := *flagMaxRSSMiB
	oldPoll := *flagRSSPoll
	*flagMaxRSSMiB = 24
	*flagRSSPoll = 10 * time.Millisecond
	defer func() {
		*flagMaxRSSMiB = oldLimit
		*flagRSSPoll = oldPoll
	}()

	_, _, _, elapsed, err := runProgram(
		t.TempDir(),
		os.Args[0],
		append(os.Environ(), "LLGO_GOROOT_HELPER=allocate"),
		5*time.Second,
		"-test.run=^TestRunProgramRSSLimit$",
	)
	if err == nil || !strings.Contains(err.Error(), "RSS") {
		t.Fatalf("err=%v, want RSS limit error", err)
	}
	if elapsed >= 5*time.Second {
		t.Fatalf("elapsed=%s, memory limit did not terminate before timeout", elapsed)
	}
}

func TestValidateSystemMemoryState(t *testing.T) {
	oldMemory := *flagMinMemPct
	oldSwap := *flagMinSwapMiB
	*flagMinMemPct = 15
	*flagMinSwapMiB = 512
	defer func() {
		*flagMinMemPct = oldMemory
		*flagMinSwapMiB = oldSwap
	}()

	if err := validateSystemMemoryState(systemMemoryState{freePercent: 20, swapFree: 1 << 30, swapPresent: true}); err != nil {
		t.Fatalf("healthy state rejected: %v", err)
	}
	if err := validateSystemMemoryState(systemMemoryState{freePercent: 10, swapFree: 1 << 30, swapPresent: true}); err == nil || !strings.Contains(err.Error(), "free memory") {
		t.Fatalf("low-memory err=%v", err)
	}
	if err := validateSystemMemoryState(systemMemoryState{freePercent: 20, swapFree: 128 << 20, swapPresent: true}); err == nil || !strings.Contains(err.Error(), "free swap") {
		t.Fatalf("low-swap err=%v", err)
	}
	if err := validateSystemMemoryState(systemMemoryState{freePercent: 20, swapPresent: false}); err != nil {
		t.Fatalf("swapless state rejected: %v", err)
	}
}

func TestRunGeneratedProgramUsesProvidedTimeout(t *testing.T) {
	disableSystemMemoryLimits(t)
	dir := t.TempDir()
	tool := filepath.Join(dir, "fake-tool.sh")
	script := `#!/bin/sh
set -eu
out=""
prev=""
for arg in "$@"; do
	if [ "$prev" = "-o" ]; then
		out="$arg"
	fi
	prev="$arg"
done
cat > "$out" <<'EOF'
#!/bin/sh
sleep 0.2
EOF
chmod +x "$out"
`
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "generated.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := caseWorkspace{rootDir: dir, workDir: dir}
	// The build helper is intentionally trivial, but process startup can be
	// heavily delayed on a loaded CI host. Keep that setup budget independent
	// from the 50ms generated-program timeout this test actually verifies.
	_, _, exitCode, _, elapsed, err := runGeneratedProgram(ws, tool, os.Environ(), "generated.go", "fake", 30*time.Second, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if !strings.Contains(err.Error(), "timed out after 50ms") {
		t.Fatalf("err=%v, exitCode=%d, want timeout mentioning 50ms", err, exitCode)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("elapsed=%s, want >= 50ms", elapsed)
	}
}

func TestNormalizeOutputStripsLogTimestamp(t *testing.T) {
	in := []byte("2026/03/13 00:56:11 listing stdlib export files: open : no such file or directory\n")
	got := string(normalizeOutput(in))
	want := "listing stdlib export files: open : no such file or directory\n"
	if got != want {
		t.Fatalf("normalizeOutput()=%q, want %q", got, want)
	}
}

func TestShardCases(t *testing.T) {
	cases := []testCase{
		{RelPath: "a.go"},
		{RelPath: "b.go"},
		{RelPath: "c.go"},
		{RelPath: "d.go"},
		{RelPath: "e.go"},
	}
	got := shardCases(t, cases, 1, 3)
	want := []testCase{
		{RelPath: "b.go"},
		{RelPath: "e.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shardCases()=%v, want %v", got, want)
	}
}

func TestDiscoverCasesSkipsMissingDir(t *testing.T) {
	testRoot := t.TempDir()
	existingDir := filepath.Join(testRoot, "fixedbugs")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(existingDir, "case.go")
	if err := os.WriteFile(file, []byte("// run\n\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := discoverCases(t, testRoot, toolchainEnv{
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		CGOEnabled: "1",
	}, []string{"fixedbugs", "internal/runtime/sys"}, nil, 0, loadDirectiveMode(t, "legacy"))
	want := []testCase{{
		RelPath:      "fixedbugs/case.go",
		Dir:          existingDir,
		FileName:     "case.go",
		Directive:    "run",
		DirectiveArg: []string{},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverCases()=%v, want %v", got, want)
	}
}

func TestDiscoverCasesRunLikeModeIncludesDirectiveArgs(t *testing.T) {
	testRoot := t.TempDir()
	dir := filepath.Join(testRoot, "fixedbugs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"run.go":       "// run -gcflags=-d=checkptr\n\npackage main\n",
		"runoutput.go": "// runoutput ./rotate.go\n\npackage main\n",
		"rundir.go":    "// rundir\n\npackage ignored\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := discoverCases(t, testRoot, toolchainEnv{
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		CGOEnabled: "1",
	}, []string{"fixedbugs"}, nil, 0, loadDirectiveMode(t, "runlike"))
	if len(got) != 3 {
		t.Fatalf("len(discoverCases())=%d, want 3", len(got))
	}
	if got[0].Directive != "rundir" && got[1].Directive != "rundir" && got[2].Directive != "rundir" {
		t.Fatalf("discoverCases()=%v, want rundir to be included", got)
	}
}

func TestDiscoverCasesCIModeExcludesBuildrundir(t *testing.T) {
	testRoot := t.TempDir()
	dir := filepath.Join(testRoot, "fixedbugs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"run.go":        "// run arg1\n\npackage main\n",
		"race.go":       "// run -race\n\npackage main\n",
		"runoutput.go":  "// runoutput\n\npackage main\n",
		"rundir.go":     "// rundir\n\npackage ignored\n",
		"runindir.go":   "// runindir\n\npackage ignored\n",
		"buildrun.go":   "// buildrun\n\npackage main\n",
		"builddir.go":   "// buildrundir\n\npackage ignored\n",
		"errorcheck.go": "// errorcheckandrundir -1\n\npackage ignored\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := discoverCases(t, testRoot, toolchainEnv{
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		CGOEnabled: "1",
	}, []string{"fixedbugs"}, nil, 0, loadDirectiveMode(t, "ci"))

	if len(got) != 3 {
		t.Fatalf("len(discoverCases())=%d, want 3", len(got))
	}
	for _, tc := range got {
		switch tc.Directive {
		case "run", "runoutput", "buildrun":
		default:
			t.Fatalf("unexpected directive in ci mode: %q", tc.Directive)
		}
		if tc.FileName == "race.go" {
			t.Fatalf("ci mode unexpectedly included -race case: %+v", tc)
		}
	}
}

func TestParseDirectiveOptions(t *testing.T) {
	opts, err := parseDirectiveOptions("runindir", []string{"-gomodversion", "1.23", "-goexperiment", "fieldtrack", "-ldflags", "-strictdups=2", "-w=0", "arg1"}, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if opts.GoModVersion != "1.23" {
		t.Fatalf("GoModVersion=%q, want 1.23", opts.GoModVersion)
	}
	if !reflect.DeepEqual(opts.ProgramArgs, []string{"arg1"}) {
		t.Fatalf("ProgramArgs=%v, want [arg1]", opts.ProgramArgs)
	}
	if !reflect.DeepEqual(opts.BuildFlags, []string{"-ldflags", "-strictdups=2 -w=0"}) {
		t.Fatalf("BuildFlags=%v", opts.BuildFlags)
	}
	if !reflect.DeepEqual(opts.ExtraEnv, []string{"GOEXPERIMENT=fieldtrack"}) {
		t.Fatalf("ExtraEnv=%v", opts.ExtraEnv)
	}
}

func TestParseCompilerDirectiveOptions(t *testing.T) {
	opts, err := parseDirectiveOptions("errorcheck", []string{
		"-0", "-lang=go1.17", "-N", "-goexperiment", "fieldtrack",
	}, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if opts.WantError {
		t.Fatal("WantError=true, want false after -0")
	}
	if !reflect.DeepEqual(opts.CompilerFlags, []string{"-lang=go1.17", "-N"}) {
		t.Fatalf("CompilerFlags=%v", opts.CompilerFlags)
	}
	if len(opts.BuildFlags) != 0 {
		t.Fatalf("BuildFlags=%v, want none", opts.BuildFlags)
	}
	if !reflect.DeepEqual(opts.ExtraEnv, []string{"GOEXPERIMENT=fieldtrack"}) {
		t.Fatalf("ExtraEnv=%v", opts.ExtraEnv)
	}
}

func TestParseRundirCompilerAndLinkerFlags(t *testing.T) {
	opts, err := parseDirectiveOptions("rundir", []string{"-N", "-ldflags", "-w", "-strictdups=2"}, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(opts.CompilerFlags, []string{"-N"}) {
		t.Fatalf("CompilerFlags=%v", opts.CompilerFlags)
	}
	if !reflect.DeepEqual(opts.LinkerFlags, []string{"-w", "-strictdups=2"}) {
		t.Fatalf("LinkerFlags=%v", opts.LinkerFlags)
	}
}

func TestDirectoryBuildFlagsMatchGoFlagShape(t *testing.T) {
	goFlags, llgoFlags := directoryBuildFlags(directiveOptions{
		CompilerFlags: []string{"-N", "-l=4"},
		LinkerFlags:   []string{"-strictdups=2", "-w=0"},
	})
	want := []string{"-gcflags=-N -l=4", "-ldflags=-strictdups=2 -w=0"}
	if !reflect.DeepEqual(goFlags, want) {
		t.Fatalf("goFlags=%v, want %v", goFlags, want)
	}
	if !reflect.DeepEqual(llgoFlags, want) {
		t.Fatalf("llgoFlags=%v, want %v", llgoFlags, want)
	}
}

func TestCheckExpectedErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "case.go")
	src := `package p
var _ = missing // ERROR "undefined: missing"
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	output := file + ":2:9: undefined: missing\n"
	if err := checkExpectedErrors(output, file, "case.go"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckExpectedErrorsReportsMissingDiagnostic(t *testing.T) {
	file := filepath.Join(t.TempDir(), "case.go")
	src := `package p
var _ = missing // ERROR "undefined: missing"
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	err := checkExpectedErrors("", file, "case.go")
	if err == nil || !strings.Contains(err.Error(), "missing error") {
		t.Fatalf("err=%v, want missing error", err)
	}
}

func TestCheckExpectedErrorsNormalizesLexicalDiagnostics(t *testing.T) {
	file := filepath.Join(t.TempDir(), "case.go")
	src := `package p
var _ = 0 // ERROR "newline in string"
var _ = 0 // ERROR "newline in rune literal"
var _ = 0 // ERROR "string not terminated"
var _ = 0 // ERROR "invalid character .* in identifier"
var _ = 0 // ERROR "identifier cannot begin with digit"
var _ = 0 // ERROR "oct|char"
var _ = 0 // ERROR "hexadecimal literal has no digits"
var _ = 0 // ERROR "empty rune literal"
var _ = 0 // ERROR "mantissa requires a 'p' exponent"
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	output := strings.Join([]string{
		file + ":2: string literal not terminated",
		file + ":2: invalid import path (invalid syntax)",
		file + ":2: malformed constant: \"",
		file + ":3: rune literal not terminated",
		file + ":3: malformed constant: '",
		file + ":4: raw string literal not terminated",
		file + ":4: expected '}', found 'EOF'",
		file + ":4: malformed constant: `",
		file + ":5: invalid character U+229B '⊛' in identifier",
		file + ":5: expected type, found 'ILLEGAL'",
		file + ":5: illegal character U+229B '⊛'",
		file + ":6: identifier cannot begin with digit U+06F6 '۶'",
		file + ":6: expected 'IDENT', found 'ILLEGAL'",
		file + ":6: illegal character U+06F6 '۶'",
		file + ":7: invalid character '\\'' in octal escape",
		file + ":7: malformed constant: '\\0'",
		file + ":8: hexadecimal literal has no digits",
		file + ":8: malformed constant: 0x",
		file + ":9: empty rune literal or unescaped '",
		file + ":9: syntax error: unexpected literal after top level declaration",
		file + ":9: illegal rune literal",
		file + ":10: hexadecimal mantissa requires a 'p' exponent",
		file + ":10: 0x1.0 (untyped float constant 1) is not used",
	}, "\n")
	if err := checkExpectedErrors(output, file, "case.go"); err != nil {
		t.Fatal(err)
	}
}

func TestCheckExpectedErrorsDiscardsPairedMissingCommaDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		line      int
		primary   string
		secondary string
	}{
		{
			name: "composite literal",
			source: `package p
var _ = []int{
	3 // ERROR "need trailing comma before newline in composite literal|possibly missing comma or }"
}
`,
			line:      3,
			primary:   "syntax error: unexpected newline in composite literal; possibly missing comma or }",
			secondary: "missing ',' before newline in composite literal",
		},
		{
			name: "parameter list",
			source: `package p
func f(x int /* // GC_ERROR "unexpected newline"

*/) // GCCGO_ERROR "expected .*\).*|expected declaration"
`,
			line:      2,
			primary:   "syntax error: unexpected newline in parameter list; possibly missing comma or )",
			secondary: "missing ',' before newline in parameter list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			output := fmt.Sprintf("%s:%d: %s\n%s:%d: %s\n", file, tt.line, tt.primary, file, tt.line, tt.secondary)
			if err := checkExpectedErrors(output, file, "case.go"); err != nil {
				t.Fatal(err)
			}
			if err := checkExpectedErrors(fmt.Sprintf("%s:%d: %s\n", file, tt.line, tt.secondary), file, "case.go"); err == nil {
				t.Fatal("secondary diagnostic passed without its primary")
			}
			wrongLine := fmt.Sprintf("%s:%d: %s\n%s:%d: %s\n", file, tt.line, tt.primary, file, tt.line+1, tt.secondary)
			if err := checkExpectedErrors(wrongLine, file, "case.go"); err == nil || !strings.Contains(err.Error(), tt.secondary) {
				t.Fatalf("secondary diagnostic on another line was discarded: %v", err)
			}
			if got := parserRecoverySecondaries(tt.primary + "."); got != nil {
				t.Fatalf("near-match primary activated parser recovery: %v", got)
			}
		})
	}
}

func TestCheckExpectedErrorsKeepsUnrelatedDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		output func(string) string
	}{
		{
			name: "same line non-recovery",
			src:  "package p\nvar _ = 0 // ERROR \"unknown escape\"\n",
			output: func(file string) string {
				return file + ":2: unknown escape\n" + file + ":2: undefined: still_reported\n"
			},
		},
		{
			name: "different line recovery",
			src:  "package p\nvar _ = 0 // ERROR \"unknown escape\"\nvar _ = 0\n",
			output: func(file string) string {
				return file + ":2: unknown escape\n" + file + ":3: malformed constant: 0x\n"
			},
		},
		{
			name: "unscoped invalid character",
			src:  "package p\nvar _ = 0 // ERROR \"invalid character U\\+003F\"\n",
			output: func(file string) string {
				return file + ":2: invalid character U+003F '?'\n" +
					file + ":2: expected 'IDENT', found 'ILLEGAL'\n" +
					file + ":2: illegal character U+003F '?'\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			err := checkExpectedErrors(tt.output(file), file, "case.go")
			if err == nil || !strings.Contains(err.Error(), "unmatched errors") {
				t.Fatalf("err=%v, want unmatched errors", err)
			}
		})
	}
}

func TestCheckExpectedErrorsDiscardsExactParserPair(t *testing.T) {
	file := filepath.Join(t.TempDir(), "case.go")
	src := "package p\nfunc f() {\n\tif a := 10 { // ERROR \"cannot use a := 10 as value\"\n\t}\n}\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	primary := file + ":3: syntax error: cannot use a := 10 as value"
	secondary := file + ":3: expected boolean expression, found assignment (missing parentheses around composite literal?)"
	if err := checkExpectedErrors(primary+"\n"+secondary, file, "case.go"); err != nil {
		t.Fatal(err)
	}
	if err := checkExpectedErrors(primary+"\n"+secondary+"\n"+file+":4: undefined: extra", file, "case.go"); err == nil || !strings.Contains(err.Error(), "undefined: extra") {
		t.Fatalf("err=%v, want unrelated diagnostic to remain", err)
	}
	for _, nearMatch := range []string{
		"syntax error: cannot use a := 10 as value.",
		"syntax error: cannot use d := 10 as value",
	} {
		if got := parserRecoverySecondaries(nearMatch); got != nil {
			t.Fatalf("near-match %q activated parser recovery: %v", nearMatch, got)
		}
	}
}

func TestCheckExpectedErrorsDiscardsPairedDeclarationRecoveryDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wrong     string
		line      int
		primary   string
		secondary string
	}{
		{
			name:      "missing package clause",
			source:    "func main() { // ERROR \"package\"\n}\n",
			wrong:     "func other() { // ERROR \"package\"\n}\n",
			line:      1,
			primary:   "expected 'package', found 'func'",
			secondary: "expected ';', found '('",
		},
		{
			name: "top-level composite literal",
			source: `package p
var x map[string]string{"a":"b"} // ERROR "unexpected { at end of statement|unexpected { after top level declaration|expected ';' or newline after top level declaration"
`,
			wrong: `package p
var y map[string]string{"a":"b"} // ERROR "unexpected { at end of statement|unexpected { after top level declaration|expected ';' or newline after top level declaration"
`,
			line:      2,
			primary:   "syntax error: unexpected { after top level declaration",
			secondary: "expected ';', found '{'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			output := fmt.Sprintf("%s:%d: %s\n%s:%d: %s\n", file, tt.line, tt.primary, file, tt.line, tt.secondary)
			if err := checkExpectedErrors(output, file, "case.go"); err != nil {
				t.Fatal(err)
			}
			if err := checkExpectedErrors(fmt.Sprintf("%s:%d: %s\n", file, tt.line, tt.secondary), file, "case.go"); err == nil {
				t.Fatal("secondary diagnostic passed without its primary")
			}
			if got := parserRecoverySecondaries(tt.primary); got != nil {
				t.Fatalf("source-dependent primary activated source-independent recovery: %v", got)
			}
			if got := parserRecoverySecondaries(tt.primary + "."); got != nil {
				t.Fatalf("near-match primary activated parser recovery: %v", got)
			}

			wrongFile := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(wrongFile, []byte(tt.wrong), 0o644); err != nil {
				t.Fatal(err)
			}
			wrongOutput := fmt.Sprintf("%s:%d: %s\n%s:%d: %s\n", wrongFile, tt.line, tt.primary, wrongFile, tt.line, tt.secondary)
			err := checkExpectedErrors(wrongOutput, wrongFile, "case.go")
			if err == nil || !strings.Contains(err.Error(), tt.secondary) {
				t.Fatalf("wrong source shape discarded secondary: %v", err)
			}
		})
	}
}

func TestCheckExpectedErrorsDiscardsAdditionalParserRecoveryDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		source string
		output func(file string) string
	}{
		{
			name: "variadic result Go 1.24",
			source: `package p
func g(x int, y float32) (...) // ERROR "[.][.][.]"
`,
			output: func(file string) string {
				return file + ":2: syntax error: ... is missing type\n" +
					file + ":2: expected type, found ')'\n"
			},
		},
		{
			name: "variadic result Go 1.25 and newer",
			source: `package p
func g(x int, y float32) (...) // ERROR "[.][.][.]"
`,
			output: func(file string) string {
				return file + ":2: syntax error: ... is missing type\n" +
					file + ":2: invalid use of ...\n" +
					file + ":2: expected type, found ')'\n"
			},
		},
		{
			name: "channel send in if condition",
			source: `package p
var c chan int
var v int
func f() {
	if c <- v { // ERROR "cannot use c <- v as value|send statement used as value"
	}
}
`,
			output: func(file string) string {
				return file + ":5: syntax error: cannot use c <- v as value\n" +
					file + ":5: expected boolean expression, found simple statement (missing parentheses around composite literal?)\n"
			},
		},
		{
			name: "channel send at top level",
			source: `package p
var c chan int
var v int
var _ = c <- v // ERROR "unexpected <-|send statement used as value"
`,
			output: func(file string) string {
				return file + ":4: syntax error: unexpected <- after top level declaration\n" +
					file + ":4: expected ';', found '<-'\n"
			},
		},
		{
			name: "illegal declaration character",
			source: `package p
var? // ERROR "invalid character U\+003F '\?'|invalid character 0x3f in input file"
`,
			output: func(file string) string {
				return file + ":2: invalid character U+003F '?'\n" +
					file + ":2: expected 'IDENT', found 'ILLEGAL'\n" +
					file + ":2: illegal character U+003F '?'\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := checkExpectedErrors(tt.output(file), file, "case.go"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckExpectedErrorsScopesVareqRecovery(t *testing.T) {
	const (
		source = `package main
func main() {
	var x map[string]string{"a":"b"} // ERROR "unexpected { at end of statement|expected ';' or '}' or newline"`
		primary = "syntax error: unexpected { at end of statement"
	)
	secondaries := []string{
		"expected ';', found '{'",
		"expected ';', found 'EOF'",
		"expected '}', found 'EOF'",
		"declared and not used: x",
	}
	check := func(t *testing.T, source string, includePrimary bool, secondaryLine int, extras ...string) error {
		t.Helper()
		file := filepath.Join(t.TempDir(), "case.go")
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		var output []string
		if includePrimary {
			output = append(output, file+":3: "+primary)
		}
		for _, secondary := range secondaries {
			output = append(output, fmt.Sprintf("%s:%d: %s", file, secondaryLine, secondary))
		}
		for _, extra := range extras {
			output = append(output, fmt.Sprintf("%s:%d: %s", file, secondaryLine, extra))
		}
		return checkExpectedErrors(strings.Join(output, "\n"), file, "case.go")
	}

	t.Run("exact source primary and line", func(t *testing.T) {
		if err := check(t, source, true, 3); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("wrong source", func(t *testing.T) {
		wrong := strings.Replace(source, `"a":"b"`, `"a":"c"`, 1)
		if err := check(t, wrong, true, 3); err == nil || !strings.Contains(err.Error(), secondaries[0]) {
			t.Fatalf("err=%v, want recovery diagnostics to remain", err)
		}
	})
	t.Run("missing primary", func(t *testing.T) {
		if err := check(t, source, false, 3); err == nil || !strings.Contains(err.Error(), "no match") {
			t.Fatalf("err=%v, want missing primary to fail", err)
		}
	})
	t.Run("wrong line", func(t *testing.T) {
		if err := check(t, source, true, 4); err == nil || !strings.Contains(err.Error(), secondaries[0]) {
			t.Fatalf("err=%v, want wrong-line recovery diagnostics to remain", err)
		}
	})
	t.Run("unrelated type diagnostic", func(t *testing.T) {
		const extra = "undefined: still_reported"
		if err := check(t, source, true, 3, extra); err == nil || !strings.Contains(err.Error(), extra) {
			t.Fatalf("err=%v, want unrelated diagnostic to remain", err)
		}
	})
	t.Run("duplicate secondary", func(t *testing.T) {
		// The complete pipeline deduplicates exact diagnostics before pairing,
		// so exercise the one-use recovery matcher directly.
		file := filepath.Join(t.TempDir(), "case.go")
		pairs := make([]parserRecoveryPair, 0, len(secondaries))
		sourceLine := strings.Split(source, "\n")[2]
		for _, group := range parserRecoverySecondaryGroups(primary, sourceLine) {
			pairs = append(pairs, parserRecoveryPair{
				file: canonicalDiagnosticPath(file), line: 3, secondaries: group,
			})
		}
		duplicate := file + ":3: declared and not used: x"
		got := discardPairedParserDiagnostics(
			[]string{duplicate, duplicate},
			newDiagnosticPathResolver([]diagnosticSource{{full: file, short: "case.go"}}),
			pairs,
		)
		if want := []string{duplicate}; !reflect.DeepEqual(got, want) {
			t.Fatalf("duplicate recovery diagnostics=%v, want %v", got, want)
		}
	})
}

func TestAdditionalParserRecoveryDiagnosticsFailOpen(t *testing.T) {
	t.Run("one ERROR authorizes one recovery group", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
func f() {
	if a := 10 { // ERROR "cannot use [ab] := 10 as value"
	}
}
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":3: syntax error: cannot use a := 10 as value\n" +
			file + ":3: syntax error: cannot use b := 10 as value\n" +
			file + ":3: expected boolean expression, found assignment (missing parentheses around composite literal?)\n" +
			file + ":3: expected boolean or range expression, found assignment (missing parentheses around composite literal?)\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected boolean or range expression") {
			t.Fatalf("err=%v, want the second matching primary's recovery diagnostic to remain", err)
		}
	})

	t.Run("missing primary", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var? // ERROR "invalid character U\+003F '\?'"
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":2: expected 'IDENT', found 'ILLEGAL'\n" +
			file + ":2: illegal character U+003F '?'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), `no match for "invalid character`) {
			t.Fatalf("err=%v, want missing primary to fail", err)
		}
	})

	t.Run("primary does not match ERROR", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var? // ERROR "different diagnostic"
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":2: invalid character U+003F '?'\n" +
			file + ":2: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want unmatched primary to leave recovery diagnostic visible", err)
		}
	})

	t.Run("same line different illegal token", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var _ = ?; var@ // ERROR "invalid character U\+003F '\?'"
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":2: invalid character U+003F '?'\n" +
			file + ":2: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want the unrelated @ recovery diagnostic to remain", err)
		}
	})

	t.Run("wrong line", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var? // ERROR "invalid character U\+003F '\?'"

`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":2: invalid character U+003F '?'\n" +
			file + ":3: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), ":3: expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want wrong-line recovery to remain", err)
		}
	})

	t.Run("near match primary", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var? // ERROR "invalid character U\+003F '\?'"
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":2: invalid character U+003F '?'.\n" +
			file + ":2: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want recovery after near-match primary to remain", err)
		}
	})

	t.Run("wrong channel source shape", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
var c chan int
var v int
func f() {
	if (c <- v) { // ERROR "cannot use c <- v as value"
	}
}
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":5: syntax error: cannot use c <- v as value\n" +
			file + ":5: expected boolean expression, found simple statement (missing parentheses around composite literal?)\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected boolean expression") {
			t.Fatalf("err=%v, want recovery for a different source shape to remain", err)
		}
	})

	t.Run("wrong file", func(t *testing.T) {
		dir := t.TempDir()
		primaryFile := filepath.Join(dir, "primary.go")
		otherFile := filepath.Join(dir, "other.go")
		if err := os.WriteFile(primaryFile, []byte(`package p
var? // ERROR "invalid character U\+003F '\?'"
`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(otherFile, []byte("package p\nvar x int\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		output := primaryFile + ":2: invalid character U+003F '?'\n" +
			otherFile + ":2: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrorsForFiles(output, []diagnosticSource{
			{full: primaryFile, short: "primary.go"},
			{full: otherFile, short: "other.go"},
		})
		if err == nil || !strings.Contains(err.Error(), "other.go:2: expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want wrong-file recovery to remain", err)
		}
	})

	t.Run("line directive", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "case.go")
		source := `package p
//line remapped.go:2
var? // ERROR "invalid character U\+003F '\?'"
`
		if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		output := file + ":3: invalid character U+003F '?'\n" +
			file + ":3: expected 'IDENT', found 'ILLEGAL'\n"
		err := checkExpectedErrors(output, file, "case.go")
		if err == nil || !strings.Contains(err.Error(), "expected 'IDENT', found 'ILLEGAL'") {
			t.Fatalf("err=%v, want line-remapped recovery to remain", err)
		}
	})
}

func TestParserRecoverySourceCode(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "URL before ERROR comment",
			source: `var _ = "http://example.com" // ERROR "broken"`,
			want:   `var _ = "http://example.com"`,
		},
		{
			name:   "URL without ERROR comment",
			source: `var _ = "http://example.com"`,
			want:   `var _ = "http://example.com"`,
		},
		{
			name:   "GC ERRORAUTO comment",
			source: `var _ = "http://example.com" // GC_ERRORAUTO "broken"`,
			want:   `var _ = "http://example.com"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parserRecoverySourceCode(tt.source); got != tt.want {
				t.Fatalf("parserRecoverySourceCode(%q)=%q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestHasLineDirective(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{name: "line comment", data: "\t//line remapped.go:1\n", want: true},
		{name: "block comment", data: "  /*line remapped.go:1*/\n", want: true},
		{name: "block comment after source", data: "x /*line remapped.go:1*/\n", want: true},
		{name: "missing separator", data: "//linefoo.go:1\n", want: false},
		{name: "line comment without line number", data: "//line remapped\n", want: false},
		{name: "block comment without line number", data: "/*line remapped*/\n", want: false},
		{name: "after source", data: "x //line remapped.go:1\n", want: false},
		{name: "inside string", data: `var _ = "/*line remapped.go:1*/"` + "\n", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasLineDirective([]byte(tt.data)); got != tt.want {
				t.Fatalf("hasLineDirective(%q)=%v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestCheckExpectedErrorsScopesImportAlias(t *testing.T) {
	tests := []struct {
		name, src string
		wantOK    bool
	}{
		{"parenthesized import", "package p\nimport (\n\t\"io\", // ERROR \"unexpected comma\"\n)\n", true},
		{"outside import", "package p\n\nvar _ = 0 // ERROR \"unexpected comma\"\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			err := checkExpectedErrors(file+":3: expected ';', found ','", file, "case.go")
			if tt.wantOK && err != nil || !tt.wantOK && (err == nil || !strings.Contains(err.Error(), "no match")) {
				t.Fatalf("err=%v, wantOK=%v", err, tt.wantOK)
			}
		})
	}
}

func TestDiscardPairedParserDiagnosticsIsExactMultiset(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a", "case.go")
	fileB := filepath.Join(dir, "b", "case.go")
	resolver := newDiagnosticPathResolver([]diagnosticSource{
		{full: fileA, short: "a.go"}, {full: fileB, short: "b.go"},
	})
	secondary := "expected boolean expression, found assignment (missing parentheses around composite literal?)"
	exact := fileA + ":2: " + secondary
	wrongLine := fileA + ":3: " + secondary
	wrongFile := fileB + ":2: " + secondary
	similar := exact + " extra"
	pairs := []parserRecoveryPair{{
		file: canonicalDiagnosticPath(fileA), line: 2, secondaries: []string{secondary},
	}}
	got := discardPairedParserDiagnostics([]string{exact, exact, wrongLine, wrongFile, similar}, resolver, pairs)
	want := []string{exact, wrongLine, wrongFile, similar}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discardPairedParserDiagnostics()=%v, want %v", got, want)
	}

	identifier := fileA + ":2: expected 'IDENT', found 'ILLEGAL'"
	illegal := fileA + ":2: illegal character U+003F '?'"
	pairs = []parserRecoveryPair{
		{file: canonicalDiagnosticPath(fileA), line: 2, secondaries: []string{"expected 'IDENT', found 'ILLEGAL'"}},
		{file: canonicalDiagnosticPath(fileA), line: 2, secondaries: []string{"illegal character U+003F '?'"}},
	}
	got = discardPairedParserDiagnostics([]string{identifier, identifier, illegal, illegal}, resolver, pairs)
	want = []string{identifier, illegal}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("independent groups=%v, want %v", got, want)
	}
}

func TestDiagnosticPathResolverRejectsUnknownAndAmbiguousSources(t *testing.T) {
	dir := t.TempDir()
	left := filepath.Join(dir, "left", "case.go")
	right := filepath.Join(dir, "right", "case.go")
	resolver := newDiagnosticPathResolver([]diagnosticSource{
		{full: left, short: "case.go"},
		{full: right, short: "case.go"},
	})
	if _, ok := resolver.resolve("case.go"); ok {
		t.Fatal("ambiguous short path resolved")
	}
	if _, ok := resolver.resolve(filepath.Join(dir, "unknown", "case.go")); ok {
		t.Fatal("unknown absolute path resolved")
	}
	if got, ok := resolver.resolve(left); !ok || got != canonicalDiagnosticPath(left) {
		t.Fatalf("known absolute path resolved to %q, %v", got, ok)
	}
}

func TestPreferSpecificDiagnostics(t *testing.T) {
	got := preferSpecificDiagnostics([]string{
		"case.go:18:16: requires go1.22 or later (file declares go1.21)",
		"/tmp/case.go:18: requires go1.22 or later",
		"case.go:19: a different error",
		"/tmp/left/case.go:20: same text",
		"/tmp/right/case.go:20: same text",
	})
	want := []string{
		"case.go:18:16: requires go1.22 or later (file declares go1.21)",
		"case.go:19: a different error",
		"/tmp/left/case.go:20: same text",
		"/tmp/right/case.go:20: same text",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preferSpecificDiagnostics()=%v, want %v", got, want)
	}
}

func TestNormalizeCompilerDiagnosticMessage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "continue", in: "continue not in for statement", want: "continue is not in a loop"},
		{name: "break", in: "break not in for, switch, or select statement", want: "break is not in a loop, switch, or select"},
		{name: "range count", in: "expected at most 2 expressions", want: "range clause permits at most two iteration variables"},
		{name: "type switch", in: "invalid syntax tree: incorrect form of type switch guard", want: "invalid variable name in type switch guard"},
		{
			name: "non-canonical import",
			in:   `non-canonical import path "unicode//utf8": should be "unicode/utf8"`,
			want: `non-canonical import path "unicode//utf8" (should be "unicode/utf8")`,
		},
		{
			name: "unexported struct field",
			in:   "cannot refer to unexported field doneChan in struct literal of type f1.Foo",
			want: "cannot refer to unexported field 'doneChan' in struct literal of type f1.Foo",
		},
		{
			name: "unknown struct field",
			in:   "unknown field DoneChan in struct literal of type f1.Foo, but does have unexported doneChan",
			want: "unknown field 'DoneChan' in struct literal of type f1.Foo (but does have unexported doneChan)",
		},
		{
			name: "unexported selector",
			in:   "f.doneChan undefined (cannot refer to unexported field doneChan)",
			want: "f.doneChan undefined (cannot refer to unexported field or method doneChan)",
		},
		{
			name: "nearby selector",
			in:   "f.name undefined (type f1.Foo has no field or method name, but does have field Name)",
			want: "f.name undefined (type f1.Foo has no field or method name, but does have Name)",
		},
		{name: "initialization", in: "initialization cycle for a", want: "initialization cycle for a"},
		{name: "goto position", in: "goto L jumps over variable declaration at line 43", want: "goto jumps over declaration"},
		{name: "unused label", in: "label L declared and not used", want: "label L defined and not used"},
		{name: "duplicate label", in: "label L already declared", want: "label L already defined"},
		{name: "missing label", in: "label L not declared", want: "label L not defined"},
		{
			name: "assignment type",
			in:   "cannot use complex(1, 2) (value of type complex128) as int value in variable declaration",
			want: "cannot use complex(1, 2) (value of type complex128) as type int in variable declaration",
		},
		{name: "different range count", in: "expected at most 3 expressions", want: "expected at most 3 expressions"},
		{name: "goto without position", in: "goto L jumps over variable declaration", want: "goto L jumps over variable declaration"},
		{name: "non-declaration assignment", in: "cannot use x as int value in argument", want: "cannot use x as int value in argument"},
		{name: "invalid import quote", in: `non-canonical import path "\x": should be "good"`, want: `non-canonical import path "\x": should be "good"`},
		{name: "non-struct field", in: "unknown field value in map literal", want: "unknown field value in map literal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCompilerDiagnosticMessage(tt.in); got != tt.want {
				t.Fatalf("normalizeCompilerDiagnosticMessage(%q)=%q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCheckExpectedErrorsFiltersDeterministicSecondaryDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		source string
		output string
	}{
		{
			name: "initialization summary",
			source: `package p
var a = b // ERROR "a refers to b\n.*b refers to a|initialization loop"
var b = a
`,
			output: "case.go:2: initialization cycle for a\ncase.go:2: a refers to b\n\tcase.go:3: b refers to a\n",
		},
		{
			name: "unused invalid label",
			source: `package p
func f() {
L:
	break L // ERROR "invalid break label L"
}
`,
			output: "case.go:4: invalid break label L\ncase.go:3: label L declared and not used\n",
		},
		{
			name: "invalid type expression",
			source: `package p
var _ = Missing // ERROR "undefined: Missing"
`,
			output: "case.go:2: undefined: Missing\ncase.go:2: Missing (type) is not an expression\n",
		},
		{
			name: "inferred array after parenthesized type",
			source: `package p
var _ = ([...]int){} // ERROR "parenthesize"
`,
			output: "case.go:2: cannot parenthesize type in composite literal\ncase.go:2: invalid use of [...] array (outside a composite literal)\n",
		},
		{
			name: "channel range arity",
			source: `package p
var _ = 0 // ERROR "range over .* permits only one iteration variable"
`,
			output: "case.go:2: range over ch (variable of type chan int) permits only one iteration variable\ncase.go:2: expected at most 2 expressions\n",
		},
		{
			name: "recursive type summary",
			source: `package p
type I interface{} // ERROR "invalid recursive type I\n\tcase.go:2:.*I refers to I"
`,
			output: "case.go:2: invalid recursive type I\ncase.go:2: invalid recursive type I\n\tcase.go:2: I refers to I\n",
		},
		{
			name: "error limit",
			source: `package p
var _ = a // ERROR "undefined: a" "too many errors"
var _ = b
`,
			output: "case.go:2: undefined: a\ncase.go:2: too many errors\ncase.go:3: undefined: b\n",
		},
		{
			name: "unused names in invalid go statement",
			source: `package p
import "fmt"
func f() {
	go func() { fmt.Println() } // ERROR "must be function call"
}
`,
			output: "case.go:4: expression in go must be function call\ncase.go:2: \"fmt\" imported and not used\n",
		},
		{
			name: "unused local in invalid go statement",
			source: `package p
func f() {
	var i int
	go func() { _ = i } // ERROR "must be function call"
}
`,
			output: "case.go:4: expression in go must be function call\ncase.go:3: declared and not used: i\n",
		},
		{
			name: "contextual integer shift",
			source: `package p
var x int
var _ = missing // ERROR "undefined: missing"
func f(s uint) {
	x = (1. << s) << (1 << s)
}
`,
			output: "case.go:3: undefined: missing\ncase.go:5: invalid operation: shifted operand (1. << s) (untyped float value) must be integer\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := checkExpectedErrors(tt.output, file, "case.go"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCheckExpectedErrorsKeepsUnrelatedTypeDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		source string
		output string
	}{
		{
			name: "unrelated",
			source: `package p
var _ = missing // ERROR "undefined: missing"
var _ = other
`,
			output: "case.go:2: undefined: missing\ncase.go:3: unrelated diagnostic\n",
		},
		{
			name: "unused import not referenced by primary",
			source: `package p
import "math"
func f() {
	go func() { fmt.Println() } // ERROR "must be function call"
}
`,
			output: "case.go:4: expression in go must be function call\ncase.go:2: \"math\" imported and not used\n",
		},
		{
			name: "same label in different functions",
			source: `package p
func first() {
L:
	break L // ERROR "invalid break label L"
}
func second() {
L:
}
`,
			output: "case.go:4: invalid break label L\ncase.go:7: label L declared and not used\n",
		},
		{
			name: "same local name in different functions",
			source: `package p
var i int
func first() {
	go func() { _ = i } // ERROR "must be function call"
}
func second() {
	var i int
}
`,
			output: "case.go:4: expression in go must be function call\ncase.go:7: declared and not used: i\n",
		},
		{
			name: "shift without integer context",
			source: `package p
var x float64
var _ = missing // ERROR "undefined: missing"
func f(s uint) {
	x = (1. << s) << (1 << s)
}
`,
			output: "case.go:3: undefined: missing\ncase.go:5: invalid operation: shifted operand (1. << s) (untyped float value) must be integer\n",
		},
		{
			name: "shift with integer in different function",
			source: `package p
var _ = missing // ERROR "undefined: missing"
func first() {
	var x int
	_ = x
}
func second(s uint) {
	x = (1. << s) << (1 << s)
}
`,
			output: "case.go:2: undefined: missing\ncase.go:8: invalid operation: shifted operand (1. << s) (untyped float value) must be integer\n",
		},
		{
			name: "shift with integer in sibling block",
			source: `package p
var _ = missing // ERROR "undefined: missing"
func f(s uint) {
	{
		var x int
		_ = x
	}
	x = (1. << s) << (1 << s)
}
`,
			output: "case.go:2: undefined: missing\ncase.go:8: invalid operation: shifted operand (1. << s) (untyped float value) must be integer\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "case.go")
			if err := os.WriteFile(file, []byte(tt.source), 0o644); err != nil {
				t.Fatal(err)
			}
			err := checkExpectedErrors(tt.output, file, "case.go")
			if err == nil || !strings.Contains(err.Error(), "unmatched errors") {
				t.Fatalf("err=%v, want unmatched error", err)
			}
		})
	}
}

func TestCheckExpectedErrorsSeparatesSameBasenameSources(t *testing.T) {
	root := t.TempDir()
	leftDir := filepath.Join(root, "left")
	rightDir := filepath.Join(root, "right")
	if err := os.MkdirAll(leftDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	left := filepath.Join(leftDir, "case.go")
	right := filepath.Join(rightDir, "case.go")
	if err := os.WriteFile(left, []byte("package p\nvar _ = left // ERROR \"undefined: left\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(right, []byte("package p\nvar _ = right // ERROR \"undefined: right\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := left + ":2: undefined: left\n" + right + ":2: undefined: right\n"
	err := checkExpectedErrorsForFiles(output, []diagnosticSource{
		{full: left, short: "left/case.go"},
		{full: right, short: "right/case.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSecondaryDiagnosticsDoNotCrossSameBasenameSources(t *testing.T) {
	root := t.TempDir()
	left := filepath.Join(root, "left", "case.go")
	right := filepath.Join(root, "right", "case.go")
	for _, file := range []string{left, right} {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	leftSource := `package p
func first() {
L:
	break L // ERROR "invalid break label L"
}
`
	rightSource := `package p
func second() {
L:
	_ = 0
}
var _ = missing // ERROR "undefined: missing"
`
	if err := os.WriteFile(left, []byte(leftSource), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(right, []byte(rightSource), 0o644); err != nil {
		t.Fatal(err)
	}
	output := left + ":4: invalid break label L\n" +
		right + ":6: undefined: missing\n" +
		right + ":3: label L declared and not used\n"
	err := checkExpectedErrorsForFiles(output, []diagnosticSource{
		{full: left, short: "left/case.go"},
		{full: right, short: "right/case.go"},
	})
	if err == nil || !strings.Contains(err.Error(), "right/case.go:3") {
		t.Fatalf("err=%v, want unmatched right/case.go diagnostic", err)
	}
}

func TestSplitSourceFiles(t *testing.T) {
	files, args := splitSourceFiles("index0.go", []string{"./index.go", "arg1", "arg2"})
	if !reflect.DeepEqual(files, []string{"index0.go", "index.go"}) {
		t.Fatalf("files=%v, want [index0.go index.go]", files)
	}
	if !reflect.DeepEqual(args, []string{"arg1", "arg2"}) {
		t.Fatalf("args=%v, want [arg1 arg2]", args)
	}
}

func TestEnsureModuleWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := ensureModuleWorkspace(dir, "llgo-goroot-runoutput", "1.14"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	want := "module llgo-goroot-runoutput\ngo 1.14\n"
	if string(data) != want {
		t.Fatalf("go.mod=%q, want %q", data, want)
	}
}

func TestRunOutputCaseGeneratesWithBaselineGoOnly(t *testing.T) {
	disableSystemMemoryLimits(t)
	if runtime.GOOS == "windows" {
		t.Skip("fake tool scripts use /bin/sh")
	}
	dir := t.TempDir()
	repoRoot := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	goroot := filepath.Join(dir, "goroot")
	if err := os.MkdirAll(goroot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goroot, "VERSION"), []byte("go1.24.11\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(dir, "test", "fixedbugs")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "case.go"), []byte("// runoutput\n\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "tools.log")
	goTool := filepath.Join(dir, "fake-go")
	llgoTool := filepath.Join(dir, "fake-llgo")
	writeRunOutputFakeTool(t, goTool, logPath, true)
	writeRunOutputFakeTool(t, llgoTool, logPath, false)

	tc := testCase{
		RelPath:   "fixedbugs/case.go",
		Dir:       srcDir,
		FileName:  "case.go",
		Directive: "runoutput",
	}
	opts := directiveOptions{Timeout: 30 * time.Second}
	if err := runOutputCase(t, repoRoot, goroot, goTool, llgoTool, tc, opts, 30*time.Second); err != nil {
		t.Fatal(err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(logData)
	if !strings.Contains(log, goTool+" run case.go") {
		t.Fatalf("fake go generator was not run; log:\n%s", log)
	}
	if strings.Contains(log, llgoTool+" run") {
		t.Fatalf("fake llgo should not run the runoutput generator; log:\n%s", log)
	}
	if !strings.Contains(log, goTool+" build -o") || !strings.Contains(log, llgoTool+" build -o") {
		t.Fatalf("generated source was not built by both tools; log:\n%s", log)
	}
}

func disableSystemMemoryLimits(t *testing.T) {
	t.Helper()
	oldMemory := *flagMinMemPct
	oldSwap := *flagMinSwapMiB
	*flagMinMemPct = 0
	*flagMinSwapMiB = 0
	t.Cleanup(func() {
		*flagMinMemPct = oldMemory
		*flagMinSwapMiB = oldSwap
	})
}

func writeRunOutputFakeTool(t *testing.T, path, logPath string, allowRun bool) {
	t.Helper()
	allowRunValue := "false"
	if allowRun {
		allowRunValue = "true"
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
printf '%%s\n' "$0 $*" >> %[1]q
case "$1" in
run)
	if [ %[2]q != "true" ]; then
		echo "unexpected runoutput generator invocation" >&2
		exit 23
	fi
	cat <<'EOF'
package main

func main() {
	print("ok\n")
}
EOF
	;;
build)
	out=""
	last=""
	prev=""
	for arg in "$@"; do
		if [ "$prev" = "-o" ]; then
			out="$arg"
		fi
		last="$arg"
		prev="$arg"
	done
	if [ -z "$out" ]; then
		echo "missing -o" >&2
		exit 24
	fi
	if [ ! -s "$last" ]; then
		echo "empty generated source: $last" >&2
		exit 25
	fi
	cat > "$out" <<'EOF'
#!/bin/sh
printf 'ok\n'
EOF
	chmod +x "$out"
	;;
*)
	echo "unexpected command: $*" >&2
	exit 26
	;;
esac
`, logPath, allowRunValue)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestToolchainGoModVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("go1.24.11\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := toolchainGoModVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.24" {
		t.Fatalf("toolchainGoModVersion()=%q, want 1.24", got)
	}
}

func TestStageRundirLayoutRewritesRelativeImports(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "a.go"), []byte("package a\nconst X = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "b.go"), []byte("package b\nimport \"./a\"\nconst Y = a.X\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nimport \"./b\"\nfunc main(){_ = b.Y}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "helper_test.go"), []byte("package main\nfunc helper() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "external.go"), []byte("package external\nfunc Missing()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dstDir := t.TempDir()
	if err := stageRundirLayout(dstDir, srcDir, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dstDir, "b", "b.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\"test/a\"") {
		t.Fatalf("rewritten file=%q, want import test/a", got)
	}
	got, err = os.ReadFile(filepath.Join(dstDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\"test/b\"") {
		t.Fatalf("rewritten file=%q, want import test/b", got)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "helper_testdir.go")); err != nil {
		t.Fatalf("stage _test.go as ordinary source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "testdir_empty.s")); err != nil {
		t.Fatalf("stage empty assembly source for missing function body: %v", err)
	}
}
