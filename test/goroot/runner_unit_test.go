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
	_, _, exitCode, _, elapsed, err := runGeneratedProgram(ws, tool, os.Environ(), "generated.go", "fake", time.Second, 50*time.Millisecond)
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

func TestPreferSpecificDiagnostics(t *testing.T) {
	got := preferSpecificDiagnostics([]string{
		"case.go:18:16: requires go1.22 or later (file declares go1.21)",
		"/tmp/case.go:18: requires go1.22 or later",
		"case.go:19: a different error",
	})
	want := []string{
		"case.go:18:16: requires go1.22 or later (file declares go1.21)",
		"case.go:19: a different error",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preferSpecificDiagnostics()=%v, want %v", got, want)
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
	opts := directiveOptions{Timeout: 5 * time.Second}
	if err := runOutputCase(t, repoRoot, goroot, goTool, llgoTool, tc, opts, 5*time.Second); err != nil {
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
