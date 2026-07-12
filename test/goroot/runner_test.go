package goroot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"go.yaml.in/yaml/v3"
)

var (
	flagGOROOT     = flag.String("goroot", os.Getenv("LLGO_GOROOT"), "Go toolchain root whose GOROOT/test sources should be used")
	flagGoCmd      = flag.String("go", os.Getenv("LLGO_GO"), "go binary used as baseline (default: <goroot>/bin/go)")
	flagLLGO       = flag.String("llgo", os.Getenv("LLGO_TEST_LLGO"), "llgo binary used for comparisons (default: build from current checkout)")
	flagDirs       = flag.String("dirs", strings.Join(defaultGoRootTestDirs, ","), "comma-separated GOROOT/test subdirectories to scan")
	flagCase       = flag.String("case", os.Getenv("LLGO_GOROOT_CASE"), "regexp selecting cases by relative path")
	flagLimit      = flag.Int("limit", 0, "maximum number of matching cases to run")
	flagShardI     = flag.Int("shard-index", 0, "0-based shard index used to partition matching cases")
	flagShardN     = flag.Int("shard-total", 1, "number of shards used to partition matching cases")
	flagKeep       = flag.Bool("keepwork", false, "keep temporary work directories for debugging")
	flagDirMode    = flag.String("directive-mode", "legacy", "case discovery mode: legacy, ci, runlike, or coverage")
	flagDirective  = flag.String("directives", "", "comma-separated directive filter within the selected mode")
	flagXFail      = flag.String("xfail", filepath.Join("test", "goroot", "xfail.yaml"), "xfail configuration path relative to repo root")
	flagBuildTO    = flag.Duration("build-timeout", 3*time.Minute, "timeout for each go/llgo build step; 0 disables the timeout")
	flagRunTO      = flag.Duration("run-timeout", time.Minute, "timeout for the compiled program run step; 0 disables the timeout")
	flagSlowBld    = flag.Duration("slow-build", 10*time.Second, "log build steps that exceed this duration; 0 disables slow-build logging")
	flagSlowRun    = flag.Duration("slow-run", 5*time.Second, "log run steps that exceed this duration; 0 disables slow-run logging")
	flagProgress   = flag.Duration("progress", 0, "log current GOROOT case progress at this interval; 0 disables progress logging")
	flagListCases  = flag.Bool("list-cases", false, "list selected case counts without building or running llgo")
	flagListPaths  = flag.Bool("list-case-paths", false, "list selected directive and case paths without building or running llgo")
	flagMaxRSSMiB  = flag.Int64("max-rss-mib", 4096, "maximum RSS in MiB for each spawned process group; 0 disables the limit")
	flagRSSWarnMiB = flag.Int64("rss-warn-mib", 1024, "log commands whose observed peak process-group RSS reaches this value; 0 disables warnings")
	flagRSSPoll    = flag.Duration("rss-poll", 100*time.Millisecond, "process-group RSS sampling interval")
	flagMinMemPct  = flag.Int("min-memory-free-percent", 15, "minimum system-wide free memory percentage required to start and continue a command; 0 disables the check")
	flagMinSwapMiB = flag.Int64("min-swap-free-mib", 512, "minimum free swap in MiB required to start and continue a command; 0 disables the check")
	flagMemPoll    = flag.Duration("memory-pressure-poll", time.Second, "system memory pressure sampling interval")
)

var defaultGoRootTestDirs = []string{
	".",
	"ken",
	"chan",
	"interface",
	"internal/runtime/sys",
	"syntax",
	"dwarf",
	"fixedbugs",
	"codegen",
	"abi",
	"typeparam",
	"typeparam/mdempsky",
	"arenas",
}

type toolchainEnv struct {
	GOOS        string
	GOARCH      string
	GOVERSION   string
	CGOEnabled  string `json:"CGO_ENABLED"`
	ReleaseTags []string
}

type testCase struct {
	RelPath      string
	Dir          string
	FileName     string
	Directive    string
	DirectiveArg []string
}

type xfailConfig struct {
	Entries   []xfailEntry   `yaml:"xfails"`
	Flakes    []xfailEntry   `yaml:"flakes"`
	HostSkips []xfailEntry   `yaml:"host_skips"`
	Timeouts  []timeoutEntry `yaml:"timeouts"`
}

type xfailEntry struct {
	Version   string `yaml:"version"`
	Platform  string `yaml:"platform"`
	Directive string `yaml:"directive"`
	Case      string `yaml:"case"`
	Reason    string `yaml:"reason"`
}

type timeoutEntry struct {
	Version   string `yaml:"version"`
	Platform  string `yaml:"platform"`
	Directive string `yaml:"directive"`
	Case      string `yaml:"case"`
	Timeout   string `yaml:"timeout"`
	Reason    string `yaml:"reason"`
}

type directiveMode struct {
	Name         string
	Directives   map[string]bool
	AllowRunArgs bool
	Recursive    bool
}

func (m directiveMode) allows(directive string, args []string) bool {
	if !m.Directives[directive] {
		return false
	}
	if directive == "run" && len(args) != 0 && !m.AllowRunArgs {
		return false
	}
	if m.Name == "ci" {
		for _, arg := range args {
			if arg == "-race" {
				return false
			}
		}
	}
	return true
}

type directiveOptions struct {
	BuildFlags     []string
	CompilerFlags  []string
	LinkerFlags    []string
	ProgramArgs    []string
	ExtraEnv       []string
	GoModVersion   string
	SingleFilePkgs bool
	WantError      bool
	Timeout        time.Duration
}

type caseMetrics struct {
	goBuild   time.Duration
	llgoBuild time.Duration
	goRun     time.Duration
	llgoRun   time.Duration
}

type caseWorkspace struct {
	rootDir string
	workDir string
	gopath  string
	cleanup func()
}

type gorootProgress struct {
	mu         sync.Mutex
	total      int
	current    int
	casePath   string
	caseStart  time.Time
	suiteStart time.Time
	done       bool
}

func startGorootProgress(total int, interval time.Duration) (*gorootProgress, func()) {
	progress := &gorootProgress{
		total:      total,
		suiteStart: time.Now(),
	}
	if interval <= 0 {
		return progress, func() {}
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				progress.Log()
			case <-stopCh:
				return
			}
		}
	}()
	return progress, func() {
		close(stopCh)
		<-doneCh
	}
}

func (p *gorootProgress) StartCase(index int, casePath string) {
	p.mu.Lock()
	p.current = index
	p.casePath = casePath
	p.caseStart = time.Now()
	p.done = false
	p.mu.Unlock()
}

func (p *gorootProgress) FinishCase(casePath string) {
	p.mu.Lock()
	if p.casePath != casePath {
		p.mu.Unlock()
		return
	}
	elapsed := time.Since(p.caseStart)
	current := p.current
	total := p.total
	p.casePath = ""
	p.caseStart = time.Time{}
	p.done = current >= total
	p.mu.Unlock()

	if *flagProgress > 0 && elapsed >= *flagProgress {
		fmt.Fprintf(os.Stderr, "goroot case %d/%d done after %s: %s\n", current, total, elapsed.Round(time.Millisecond), casePath)
	}
}

func (p *gorootProgress) Log() {
	p.mu.Lock()
	current := p.current
	total := p.total
	casePath := p.casePath
	caseElapsed := time.Since(p.caseStart)
	totalElapsed := time.Since(p.suiteStart)
	done := p.done
	p.mu.Unlock()

	if done {
		return
	}
	if casePath == "" {
		fmt.Fprintf(os.Stderr, "goroot runner progress: waiting for next case after %s\n", totalElapsed.Round(time.Second))
		return
	}
	fmt.Fprintf(os.Stderr, "goroot runner progress: case %d/%d running for %s, total %s: %s\n", current, total, caseElapsed.Round(time.Second), totalElapsed.Round(time.Second), casePath)
}

func TestGoRootRunCases(t *testing.T) {
	if *flagGOROOT == "" {
		t.Skip("set -goroot or LLGO_GOROOT to run external GOROOT/test cases")
	}

	repoRoot := repoRoot(t)
	goroot, err := filepath.Abs(*flagGOROOT)
	if err != nil {
		t.Fatalf("resolve goroot: %v", err)
	}
	goCmd := *flagGoCmd
	if goCmd == "" {
		goCmd = filepath.Join(goroot, "bin", "go")
	}
	if _, err := os.Stat(goCmd); err != nil {
		t.Fatalf("stat go command %q: %v", goCmd, err)
	}

	envInfo := loadToolchainEnv(t, goCmd)
	testRoot := filepath.Join(goroot, "test")
	info, err := os.Stat(testRoot)
	if err != nil {
		t.Fatalf("stat GOROOT/test root %q: %v", testRoot, err)
	}
	if !info.IsDir() {
		t.Fatalf("GOROOT/test root %q is not a directory", testRoot)
	}

	xfails := loadXFailConfig(t, repoRoot, *flagXFail)
	caseFilter := compileCaseFilter(t, *flagCase)
	mode := loadDirectiveMode(t, *flagDirMode)
	mode = filterDirectiveMode(t, mode, *flagDirective)
	cases := discoverCases(t, testRoot, envInfo, parseDirs(*flagDirs), caseFilter, *flagLimit, mode)
	if len(cases) == 0 {
		t.Fatalf("no matching cases found under %s for directive mode %q", testRoot, mode.Name)
	}
	cases = shardCases(t, cases, *flagShardI, *flagShardN)
	if len(cases) == 0 {
		t.Skipf("no matching cases selected for shard %d/%d", *flagShardI, *flagShardN)
	}
	if *flagListCases || *flagListPaths {
		counts := make(map[string]int)
		for _, tc := range cases {
			counts[tc.Directive]++
		}
		directives := make([]string, 0, len(counts))
		for directive := range counts {
			directives = append(directives, directive)
		}
		sort.Strings(directives)
		fmt.Printf("total=%d\n", len(cases))
		for _, directive := range directives {
			fmt.Printf("%s=%d\n", directive, counts[directive])
		}
		if *flagListPaths {
			for _, tc := range cases {
				fmt.Printf("%s\t%s\n", tc.Directive, tc.RelPath)
			}
		}
		return
	}
	t.Setenv("STDLIB_IMPORTCFG", writeStdlibImportCfg(t, goCmd))

	llgoBin := *flagLLGO
	if llgoBin == "" {
		llgoBin = buildLLGOBinary(t, repoRoot, goCmd)
	}

	fmt.Fprintf(os.Stderr, "goroot=%s goversion=%s goos=%s goarch=%s shard=%d/%d cases=%d directive_mode=%s\n", goroot, envInfo.GOVERSION, envInfo.GOOS, envInfo.GOARCH, *flagShardI, *flagShardN, len(cases), mode.Name)
	progress, stopProgress := startGorootProgress(len(cases), *flagProgress)
	defer stopProgress()
	for i, tc := range cases {
		tc := tc
		t.Run(tc.RelPath, func(t *testing.T) {
			progress.StartCase(i+1, tc.RelPath)
			defer progress.FinishCase(tc.RelPath)
			if match, reason := xfails.MatchHostSkip(envInfo.GOVERSION, runtime.GOOS+"/"+runtime.GOARCH, tc); match {
				t.Skipf("skipping host-unsafe case: %s", reason)
			}
			runTimeout := *flagRunTO
			if timeout, reason, ok := xfails.MatchTimeout(envInfo.GOVERSION, envInfo.GOOS+"/"+envInfo.GOARCH, tc); ok {
				runTimeout = timeout
				t.Logf("using timeout override %s: %s", timeout, reason)
			}
			buildTimeout := effectiveBuildTimeout(*flagBuildTO, runTimeout)
			err := runCase(t, repoRoot, goroot, goCmd, llgoBin, tc, buildTimeout, runTimeout)
			var resourceErr *resourceLimitError
			if errors.As(err, &resourceErr) {
				t.Fatalf("resource guard stopped case: %v", err)
			}
			match, reason := xfails.Match(envInfo.GOVERSION, envInfo.GOOS+"/"+envInfo.GOARCH, tc)
			flaky, flakyReason := xfails.MatchFlaky(envInfo.GOVERSION, envInfo.GOOS+"/"+envInfo.GOARCH, tc)
			switch {
			case err == nil && match:
				t.Fatalf("unexpected success for xfail case: %s", reason)
			case err == nil && flaky:
				t.Logf("flaky case passed: %s", flakyReason)
			case err != nil && match:
				t.Logf("expected failure: %s", reason)
			case err != nil && flaky:
				t.Logf("known flaky failure: %s", flakyReason)
			case err != nil:
				t.Fatal(err)
			}
		})
	}
}

func writeStdlibImportCfg(t *testing.T, goCmd string) string {
	t.Helper()
	cmd := exec.Command(goCmd, "list", "-export", "-f", "{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}", "std")
	cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list stdlib exports with %s: %v\n%s", goCmd, err, output)
	}
	filePath := filepath.Join(t.TempDir(), "stdlib-importcfg")
	if err := os.WriteFile(filePath, output, 0o644); err != nil {
		t.Fatalf("write stdlib importcfg: %v", err)
	}
	return filePath
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func loadToolchainEnv(t *testing.T, goCmd string) toolchainEnv {
	t.Helper()
	cmd := exec.Command(goCmd, "env", "-json", "GOOS", "GOARCH", "GOVERSION", "CGO_ENABLED")
	cmd.Env = append(os.Environ(), "GOENV=off", "GOFLAGS=")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s env failed: %v\nstdout:\n%s\nstderr:\n%s", goCmd, err, stdout.Bytes(), stderr.Bytes())
	}
	var info toolchainEnv
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatalf("decode %s env output: %v\nstdout:\n%s\nstderr:\n%s", goCmd, err, stdout.Bytes(), stderr.Bytes())
	}
	info.ReleaseTags = releaseTagsFor(info.GOVERSION)
	return info
}

func buildLLGOBinary(t *testing.T, repoRoot, goCmd string) string {
	t.Helper()
	start := time.Now()
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "llgo")
	if runtime.GOOS == "windows" {
		outPath += ".exe"
	}
	fmt.Fprintf(os.Stderr, "building llgo test binary: %s build -tags=dev -o %s ./cmd/llgo\n", goCmd, outPath)
	cmd := exec.Command(goCmd, "build", "-tags=dev", "-o", outPath, "./cmd/llgo")
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("build llgo failed: %v\n%s", err, out.Bytes())
	}
	fmt.Fprintf(os.Stderr, "built llgo test binary in %s: %s\n", time.Since(start).Round(time.Millisecond), outPath)
	return outPath
}

func loadXFailConfig(t *testing.T, repoRoot, relPath string) xfailConfig {
	t.Helper()
	path := relPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, relPath)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return xfailConfig{}
	}
	if err != nil {
		t.Fatalf("read xfail file %q: %v", path, err)
	}
	var cfg xfailConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse xfail file %q: %v", path, err)
	}
	return cfg
}

func compileCaseFilter(t *testing.T, expr string) *regexp.Regexp {
	t.Helper()
	if expr == "" {
		return nil
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		t.Fatalf("compile case regexp %q: %v", expr, err)
	}
	return re
}

func parseDirs(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func loadDirectiveMode(t *testing.T, name string) directiveMode {
	t.Helper()
	switch name {
	case "legacy":
		return directiveMode{
			Name: "legacy",
			Directives: map[string]bool{
				"run": true,
			},
			AllowRunArgs: false,
		}
	case "ci":
		return directiveMode{
			Name: "ci",
			Directives: map[string]bool{
				"run":       true,
				"runoutput": true,
				"buildrun":  true,
			},
			AllowRunArgs: true,
		}
	case "runlike":
		return directiveMode{
			Name: "runlike",
			Directives: map[string]bool{
				"run":         true,
				"runoutput":   true,
				"rundir":      true,
				"runindir":    true,
				"buildrun":    true,
				"buildrundir": true,
			},
			AllowRunArgs: true,
		}
	case "coverage":
		return directiveMode{
			Name: "coverage",
			Directives: map[string]bool{
				"compile":             true,
				"errorcheck":          true,
				"errorcheckandrundir": true,
				"run":                 true,
				"runoutput":           true,
				"rundir":              true,
				"runindir":            true,
				"buildrun":            true,
				"buildrundir":         true,
			},
			AllowRunArgs: true,
			Recursive:    true,
		}
	default:
		t.Fatalf("unknown -directive-mode=%q", name)
		return directiveMode{}
	}
}

func filterDirectiveMode(t *testing.T, mode directiveMode, csv string) directiveMode {
	t.Helper()
	if strings.TrimSpace(csv) == "" {
		return mode
	}
	selected := make(map[string]bool)
	for _, directive := range parseDirs(csv) {
		if !mode.Directives[directive] {
			t.Fatalf("directive %q is not enabled by -directive-mode=%s", directive, mode.Name)
		}
		selected[directive] = true
	}
	mode.Directives = selected
	return mode
}

func discoverCases(t *testing.T, testRoot string, envInfo toolchainEnv, dirs []string, filter *regexp.Regexp, limit int, mode directiveMode) []testCase {
	t.Helper()
	ctx := build.Default
	ctx.GOOS = envInfo.GOOS
	ctx.GOARCH = envInfo.GOARCH
	ctx.CgoEnabled = envInfo.CGOEnabled == "1"
	ctx.GOROOT = filepath.Dir(testRoot)
	ctx.ReleaseTags = envInfo.ReleaseTags

	if mode.Recursive {
		dirs = recursiveTestDirs(t, testRoot)
	}
	var cases []testCase
	for _, relDir := range dirs {
		absDir := filepath.Join(testRoot, filepath.FromSlash(relDir))
		entries, err := os.ReadDir(absDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Logf("skipping missing GOROOT/test dir %s", absDir)
				continue
			}
			t.Fatalf("read %s: %v", absDir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".go") {
				continue
			}
			match, err := ctx.MatchFile(absDir, name)
			if err != nil || !match {
				continue
			}
			pathInTest := name
			if relDir != "." {
				pathInTest = path.Join(relDir, name)
			}
			if filter != nil && !filter.MatchString(pathInTest) {
				continue
			}
			directive, args, ok := parseDirective(filepath.Join(absDir, name))
			if !ok || !mode.allows(directive, args) {
				continue
			}
			cases = append(cases, testCase{
				RelPath:      pathInTest,
				Dir:          absDir,
				FileName:     name,
				Directive:    directive,
				DirectiveArg: args,
			})
			if limit > 0 && len(cases) >= limit {
				return cases
			}
		}
	}
	return cases
}

func recursiveTestDirs(t *testing.T, testRoot string) []string {
	t.Helper()
	dirs := []string{"."}
	err := filepath.WalkDir(testRoot, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if current == testRoot || !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "testdata" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(testRoot, current)
		if err != nil {
			return err
		}
		dirs = append(dirs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk GOROOT/test directories: %v", err)
	}
	sort.Strings(dirs)
	return dirs
}

func shardCases(t *testing.T, cases []testCase, shardIndex, shardTotal int) []testCase {
	t.Helper()
	if shardTotal < 1 {
		t.Fatalf("invalid -shard-total=%d; want >= 1", shardTotal)
	}
	if shardIndex < 0 || shardIndex >= shardTotal {
		t.Fatalf("invalid -shard-index=%d for -shard-total=%d", shardIndex, shardTotal)
	}
	if shardTotal == 1 {
		return cases
	}
	selected := make([]testCase, 0, len(cases)/shardTotal+1)
	for i, tc := range cases {
		if i%shardTotal == shardIndex {
			selected = append(selected, tc)
		}
	}
	return selected
}

func parseDirective(filePath string) (string, []string, bool) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", nil, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			break
		}
		if !strings.HasPrefix(line, "//") {
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(line, "//"))
		fields, err := splitQuoted(text)
		if err != nil {
			return "", nil, false
		}
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "run", "runoutput", "compile", "errorcheck", "errorcheckandrundir", "rundir", "runindir", "buildrun", "buildrundir":
			return fields[0], fields[1:], true
		}
	}
	return "", nil, false
}

func runCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, buildTimeout, runTimeout time.Duration) error {
	t.Helper()
	opts, err := parseDirectiveOptions(tc.Directive, tc.DirectiveArg, runTimeout)
	if err != nil {
		return err
	}
	switch tc.Directive {
	case "compile":
		return runCompileCase(t, repoRoot, goroot, llgoBin, tc, opts, buildTimeout)
	case "errorcheck":
		return runErrorCheckCase(t, repoRoot, goroot, llgoBin, tc, opts, buildTimeout)
	case "errorcheckandrundir":
		return runErrorCheckAndRunCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, buildTimeout)
	case "run", "buildrun":
		return runSingleFileCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, buildTimeout)
	case "runoutput":
		return runOutputCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, buildTimeout)
	case "rundir":
		return runDirCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, false, buildTimeout)
	case "runindir":
		return runInDirCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, buildTimeout)
	case "buildrundir":
		return runDirCase(t, repoRoot, goroot, goCmd, llgoBin, tc, opts, true, buildTimeout)
	default:
		return fmt.Errorf("unsupported directive %q", tc.Directive)
	}
}

func effectiveBuildTimeout(defaultBuildTimeout, caseTimeout time.Duration) time.Duration {
	if caseTimeout > defaultBuildTimeout {
		return caseTimeout
	}
	return defaultBuildTimeout
}

func prepareCaseWorkspace(repoRoot string) (caseWorkspace, error) {
	root, err := os.MkdirTemp("", "llgo-goroot-*")
	if err != nil {
		return caseWorkspace{}, err
	}
	gopath := filepath.Join(root, "gopath")
	llgoPath := filepath.Join(gopath, "src", "github.com", "goplus")
	if err := os.MkdirAll(llgoPath, 0o755); err != nil {
		_ = os.RemoveAll(root)
		return caseWorkspace{}, err
	}
	linkPath := filepath.Join(llgoPath, "llgo")
	if err := os.Symlink(repoRoot, linkPath); err != nil && !errors.Is(err, os.ErrExist) {
		_ = os.RemoveAll(root)
		return caseWorkspace{}, fmt.Errorf("symlink %q -> %q: %w", linkPath, repoRoot, err)
	}
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		_ = os.RemoveAll(root)
		return caseWorkspace{}, err
	}
	return caseWorkspace{
		rootDir: root,
		workDir: workDir,
		gopath:  gopath,
		cleanup: func() { _ = os.RemoveAll(root) },
	}, nil
}

func runnerEnv(repoRoot, goroot, gopath string, extra []string) []string {
	env := append([]string{}, os.Environ()...)
	pathFound := false
	for i, item := range env {
		switch {
		case strings.HasPrefix(item, "GOROOT="):
			env[i] = "GOROOT=" + goroot
		case strings.HasPrefix(item, "GOENV="):
			env[i] = "GOENV=off"
		case strings.HasPrefix(item, "GOFLAGS="):
			env[i] = "GOFLAGS="
		case strings.HasPrefix(item, "LLGO_ROOT="):
			env[i] = "LLGO_ROOT=" + repoRoot
		case strings.HasPrefix(item, "GOPATH="):
			env[i] = "GOPATH=" + gopath
		case strings.HasPrefix(item, "GO111MODULE="):
			env[i] = "GO111MODULE=off"
		case strings.HasPrefix(item, "PATH="):
			pathFound = true
			env[i] = "PATH=" + filepath.Join(goroot, "bin") + string(os.PathListSeparator) + strings.TrimPrefix(item, "PATH=")
		}
	}
	if !pathFound {
		env = append(env, "PATH="+filepath.Join(goroot, "bin"))
	}
	env = appendIfMissing(env, "GOROOT="+goroot)
	env = appendIfMissing(env, "GOENV=off")
	env = appendIfMissing(env, "GOFLAGS=")
	env = appendIfMissing(env, "LLGO_ROOT="+repoRoot)
	env = appendIfMissing(env, "GOPATH="+gopath)
	env = appendIfMissing(env, "GO111MODULE=off")
	for _, kv := range extra {
		env = upsertEnv(env, kv)
	}
	return env
}

func appendIfMissing(env []string, kv string) []string {
	key := strings.SplitN(kv, "=", 2)[0] + "="
	for _, item := range env {
		if strings.HasPrefix(item, key) {
			return env
		}
	}
	return append(env, kv)
}

func upsertEnv(env []string, kv string) []string {
	key := strings.SplitN(kv, "=", 2)[0] + "="
	for i, item := range env {
		if strings.HasPrefix(item, key) {
			env[i] = kv
			return env
		}
	}
	return append(env, kv)
}

func restoreProcessEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	if value, ok := os.LookupEnv(key); ok {
		out = append(out, prefix+value)
	}
	return out
}

func runProgram(dir, app string, env []string, timeout time.Duration, args ...string) ([]byte, []byte, int, time.Duration, error) {
	start := time.Now()
	if err := checkSystemMemoryPressure(); err != nil {
		return nil, nil, 0, time.Since(start), err
	}
	cmd := exec.Command(app, args...)
	configureProcessGroup(cmd)
	cmd.Dir = dir
	cmd.Env = upsertEnv(append([]string{}, env...), "PWD="+dir)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, nil, 0, time.Since(start), err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	var err error
	var terminationErr error
	var timeoutTimer *time.Timer
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timeoutTimer = time.NewTimer(timeout)
		timeoutCh = timeoutTimer.C
		defer timeoutTimer.Stop()
	}
	var rssTicker *time.Ticker
	var rssCh <-chan time.Time
	if (*flagMaxRSSMiB > 0 || *flagRSSWarnMiB > 0) && resourceMonitoringSupported() {
		poll := *flagRSSPoll
		if poll <= 0 {
			poll = 100 * time.Millisecond
		}
		rssTicker = time.NewTicker(poll)
		rssCh = rssTicker.C
		defer rssTicker.Stop()
	}
	var memoryTicker *time.Ticker
	var memoryCh <-chan time.Time
	if systemMemoryMonitoringSupported() && (*flagMinMemPct > 0 || *flagMinSwapMiB > 0) {
		poll := *flagMemPoll
		if poll <= 0 {
			poll = time.Second
		}
		memoryTicker = time.NewTicker(poll)
		memoryCh = memoryTicker.C
		defer memoryTicker.Stop()
	}
	var peakRSS uint64
	for {
		select {
		case err = <-waitCh:
			goto finished
		case <-timeoutCh:
			terminationErr = fmt.Errorf("timed out after %s", timeout)
			killProcessTree(cmd)
			err = <-waitCh
			goto finished
		case <-rssCh:
			rssBytes, rssErr := processGroupRSS(cmd.Process.Pid)
			if rssErr != nil {
				continue
			}
			if rssBytes > peakRSS {
				peakRSS = rssBytes
			}
			limitBytes := uint64(*flagMaxRSSMiB) << 20
			if *flagMaxRSSMiB > 0 && rssBytes > limitBytes {
				terminationErr = &resourceLimitError{message: fmt.Sprintf("process-group RSS %s exceeded limit %s", formatBytes(rssBytes), formatBytes(limitBytes))}
				killProcessTree(cmd)
				err = <-waitCh
				goto finished
			}
		case <-memoryCh:
			if memoryErr := checkSystemMemoryPressure(); memoryErr != nil {
				terminationErr = memoryErr
				killProcessTree(cmd)
				err = <-waitCh
				goto finished
			}
		}
	}

finished:
	if *flagRSSWarnMiB > 0 && peakRSS >= uint64(*flagRSSWarnMiB)<<20 {
		fmt.Fprintf(os.Stderr, "goroot resource warning: %s peak process-group RSS %s\n", filepath.Base(app), formatBytes(peakRSS))
	}
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			exitCode = exitErr.ExitCode()
		default:
			return nil, nil, 0, time.Since(start), err
		}
	}
	elapsed := time.Since(start)
	if terminationErr != nil {
		return stdout.Bytes(), stderr.Bytes(), exitCode, elapsed, terminationErr
	}
	return stdout.Bytes(), stderr.Bytes(), exitCode, elapsed, nil
}

func formatBytes(bytes uint64) string {
	const mib = uint64(1 << 20)
	return fmt.Sprintf("%.1f MiB", float64(bytes)/float64(mib))
}

type resourceLimitError struct {
	message string
}

func (err *resourceLimitError) Error() string { return err.message }

type systemMemoryState struct {
	freePercent int
	swapFree    uint64
	swapPresent bool
}

func checkSystemMemoryPressure() error {
	if !systemMemoryMonitoringSupported() || (*flagMinMemPct <= 0 && *flagMinSwapMiB <= 0) {
		return nil
	}
	state, err := readSystemMemoryState()
	if err != nil {
		return &resourceLimitError{message: fmt.Sprintf("read system memory pressure: %v", err)}
	}
	return validateSystemMemoryState(state)
}

func validateSystemMemoryState(state systemMemoryState) error {
	if *flagMinMemPct > 0 && state.freePercent < *flagMinMemPct {
		return &resourceLimitError{message: fmt.Sprintf("system free memory %d%% is below minimum %d%%", state.freePercent, *flagMinMemPct)}
	}
	if *flagMinSwapMiB > 0 && state.swapPresent {
		minimum := uint64(*flagMinSwapMiB) << 20
		if state.swapFree < minimum {
			return &resourceLimitError{message: fmt.Sprintf("free swap %s is below minimum %s", formatBytes(state.swapFree), formatBytes(minimum))}
		}
	}
	return nil
}

func requireSuccessfulExit(err error, exitCode int) error {
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("process exited with code %d", exitCode)
	}
	return nil
}

func commandFailure(prefix string, elapsed time.Duration, err error, stdout, stderr []byte, exitCode int) error {
	var msg strings.Builder
	fmt.Fprintf(&msg, "%s failed: %v", prefix, err)
	fmt.Fprintf(&msg, "\nduration: %s", elapsed.Round(time.Millisecond))
	if exitCode != 0 {
		fmt.Fprintf(&msg, "\nexit code: %d", exitCode)
	}
	if len(stdout) != 0 {
		fmt.Fprintf(&msg, "\nstdout:\n%s", normalizeOutput(stdout))
	}
	if len(stderr) != 0 {
		fmt.Fprintf(&msg, "\nstderr:\n%s", normalizeOutput(stderr))
	}
	return &commandFailureError{message: msg.String(), cause: err}
}

type commandFailureError struct {
	message string
	cause   error
}

func (err *commandFailureError) Error() string { return err.message }
func (err *commandFailureError) Unwrap() error { return err.cause }

func logSlowCase(t *testing.T, casePath string, goBuildDur, llgoBuildDur, goRunDur, llgoRunDur time.Duration) {
	t.Helper()
	slowBuild := *flagSlowBld > 0 && (goBuildDur >= *flagSlowBld || llgoBuildDur >= *flagSlowBld)
	slowRun := *flagSlowRun > 0 && (goRunDur >= *flagSlowRun || llgoRunDur >= *flagSlowRun)
	if !slowBuild && !slowRun {
		return
	}
	fmt.Fprintf(
		os.Stderr,
		"slow goroot case %s: go build=%s llgo build=%s go run=%s llgo run=%s\n",
		casePath,
		goBuildDur.Round(time.Millisecond),
		llgoBuildDur.Round(time.Millisecond),
		goRunDur.Round(time.Millisecond),
		llgoRunDur.Round(time.Millisecond),
	)
}

func normalizeOutput(in []byte) []byte {
	in = bytes.ReplaceAll(in, []byte("\r\n"), []byte("\n"))
	in = bytes.ReplaceAll(in, []byte("\r"), []byte("\n"))
	lines := bytes.SplitAfter(in, []byte{'\n'})
	if len(lines) == 0 {
		return in
	}
	var out bytes.Buffer
	for _, line := range lines {
		out.WriteString(trimLogTimestampPrefix(string(line)))
	}
	return out.Bytes()
}

func filterNoise(in []byte) []byte {
	lines := bytes.SplitAfter(in, []byte{'\n'})
	if len(lines) == 0 {
		return in
	}
	var out bytes.Buffer
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		switch {
		case strings.HasPrefix(trimmed, "WARNING: Using LLGO root for devel:"):
			continue
		case strings.HasPrefix(trimmed, "WARNING: LLGO_ROOT is not a valid LLGO root:"):
			continue
		case strings.HasPrefix(trimmed, "ld64.lld: warning:"):
			continue
		case strings.HasPrefix(trimmed, "ld.lld: warning:"):
			continue
		case strings.HasPrefix(trimmed, "ld: warning:"):
			continue
		case strings.Contains(trimmed, "could not import ") && strings.Contains(trimmed, "(no metadata for "):
			continue
		}
		out.Write(line)
	}
	return out.Bytes()
}

type wantedError struct {
	regexp  *regexp.Regexp
	pattern string
	prefix  string
	file    string
	line    int
	auto    bool
}

type diagnosticSource struct {
	full  string
	short string
}

var (
	errorCommentRx = regexp.MustCompile(`// (?:GC_)?ERROR (.*)`)
	errorAutoRx    = regexp.MustCompile(`// (?:GC_)?ERRORAUTO (.*)`)
	errorQuotesRx  = regexp.MustCompile(`"([^"]*)"`)
	errorLineRx    = regexp.MustCompile(`LINE(([+-])(\d+))?`)
	diagnosticRx   = regexp.MustCompile(`^(.*?):([0-9]+)(?::[0-9]+)?: (.*)$`)
)

func checkExpectedErrors(output, fullPath, shortPath string) error {
	return checkExpectedErrorsForFiles(output, []diagnosticSource{{full: fullPath, short: shortPath}})
}

func checkExpectedErrorsForFiles(output string, sources []diagnosticSource) error {
	lines := splitCompilerOutput(output, false)
	for _, source := range sources {
		for i := range lines {
			lines[i] = replaceDiagnosticPrefix(lines[i], source.full, source.short)
		}
	}
	lines = preferSpecificDiagnostics(lines)
	var wanted []wantedError
	for _, source := range sources {
		expected, err := wantedErrors(source.full, source.short)
		if err != nil {
			return err
		}
		wanted = append(wanted, expected...)
	}
	var errs []error
	for _, expected := range wanted {
		var candidates []string
		if expected.auto {
			candidates, lines = partitionCompilerOutput("<autogenerated>", lines)
		} else {
			candidates, lines = partitionCompilerOutput(expected.prefix, lines)
		}
		if len(candidates) == 0 {
			errs = append(errs, fmt.Errorf("%s:%d: missing error %q", expected.file, expected.line, expected.pattern))
			continue
		}
		matched := false
		for _, candidate := range candidates {
			message := candidate
			if _, suffix, ok := strings.Cut(message, " "); ok {
				message = suffix
			}
			if expected.regexp.MatchString(message) {
				matched = true
			} else {
				lines = append(lines, candidate)
			}
		}
		if !matched {
			errs = append(errs, fmt.Errorf("%s:%d: no match for %q", expected.file, expected.line, expected.pattern))
		}
	}

	local := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, ": \t") {
			continue
		}
		for _, source := range sources {
			if hasDiagnosticPrefix(line, source.full) || hasDiagnosticPrefix(line, source.short) {
				local = append(local, line)
				break
			}
		}
	}
	if len(local) != 0 {
		errs = append(errs, fmt.Errorf("unmatched errors:\n%s", strings.Join(local, "\n")))
	}
	return errors.Join(errs...)
}

func preferSpecificDiagnostics(lines []string) []string {
	discard := make([]bool, len(lines))
	type parsedDiagnostic struct {
		location string
		message  string
	}
	parsed := make([]parsedDiagnostic, len(lines))
	for i, line := range lines {
		match := diagnosticRx.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		parsed[i] = parsedDiagnostic{
			location: filepath.Base(match[1]) + ":" + match[2],
			message:  match[3],
		}
	}
	for i := range lines {
		if parsed[i].location == "" || discard[i] {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if parsed[i].location != parsed[j].location || discard[j] {
				continue
			}
			switch {
			case parsed[i].message == parsed[j].message:
				discard[j] = true
			case strings.HasPrefix(parsed[j].message, parsed[i].message):
				discard[i] = true
			case strings.HasPrefix(parsed[i].message, parsed[j].message):
				discard[j] = true
			}
		}
	}
	out := lines[:0]
	for i, line := range lines {
		if !discard[i] {
			out = append(out, line)
		}
	}
	return out
}

func splitCompilerOutput(output string, wantAuto bool) []string {
	var out []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSuffix(line, "\r")
		switch {
		case strings.HasPrefix(line, "\t") && len(out) != 0:
			out[len(out)-1] += "\n" + line
		case strings.HasPrefix(line, "go tool"), strings.HasPrefix(line, "#"):
		case !wantAuto && strings.HasPrefix(line, "<autogenerated>"):
		case strings.TrimSpace(line) != "":
			out = append(out, line)
		}
	}
	return out
}

func wantedErrors(fullPath, shortPath string) ([]wantedError, error) {
	src, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	cache := make(map[string]*regexp.Regexp)
	var out []wantedError
	for index, sourceLine := range strings.Split(string(src), "\n") {
		lineNumber := index + 1
		if strings.Contains(sourceLine, "////") {
			continue
		}
		auto := false
		match := errorAutoRx.FindStringSubmatch(sourceLine)
		if match != nil {
			auto = true
		} else {
			match = errorCommentRx.FindStringSubmatch(sourceLine)
		}
		if match == nil {
			continue
		}
		quoted := errorQuotesRx.FindAllStringSubmatch(match[1], -1)
		if len(quoted) == 0 {
			return nil, fmt.Errorf("%s:%d: invalid ERROR comment", shortPath, lineNumber)
		}
		for _, item := range quoted {
			pattern := errorLineRx.ReplaceAllStringFunc(item[1], func(marker string) string {
				line := lineNumber
				if strings.HasPrefix(marker, "LINE+") {
					delta, _ := strconv.Atoi(marker[5:])
					line += delta
				} else if strings.HasPrefix(marker, "LINE-") {
					delta, _ := strconv.Atoi(marker[5:])
					line -= delta
				}
				return fmt.Sprintf("%s:%d", shortPath, line)
			})
			re := cache[pattern]
			if re == nil {
				re, err = regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: invalid ERROR regexp %q: %w", shortPath, lineNumber, pattern, err)
				}
				cache[pattern] = re
			}
			out = append(out, wantedError{
				regexp:  re,
				pattern: pattern,
				prefix:  fmt.Sprintf("%s:%d", shortPath, lineNumber),
				file:    shortPath,
				line:    lineNumber,
				auto:    auto,
			})
		}
	}
	return out, nil
}

func hasDiagnosticPrefix(line, prefix string) bool {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return false
	}
	slash := strings.LastIndex(line[:colon], "/")
	base := line[slash+1:]
	if len(base) <= len(prefix) || !strings.HasPrefix(base, prefix) {
		return false
	}
	return base[len(prefix)] == ':' || base[len(prefix)] == '['
}

func partitionCompilerOutput(prefix string, lines []string) (matched, unmatched []string) {
	for _, line := range lines {
		if hasDiagnosticPrefix(line, prefix) {
			matched = append(matched, line)
		} else {
			unmatched = append(unmatched, line)
		}
	}
	return matched, unmatched
}

func replaceDiagnosticPrefix(value, old, new string) string {
	if !strings.Contains(value, old) {
		return value
	}
	value = strings.ReplaceAll(value, " "+old, " "+new)
	value = strings.ReplaceAll(value, "\n"+old, "\n"+new)
	value = strings.ReplaceAll(value, "\n\t"+old, "\n\t"+new)
	if strings.HasPrefix(value, old) {
		value = new + value[len(old):]
	}
	return value
}

func trimLogTimestampPrefix(line string) string {
	if len(line) < 20 {
		return line
	}
	if line[4] != '/' || line[7] != '/' || line[10] != ' ' || line[13] != ':' || line[16] != ':' || line[19] != ' ' {
		return line
	}
	for _, pos := range []int{0, 1, 2, 3, 5, 6, 8, 9, 11, 12, 14, 15, 17, 18} {
		if line[pos] < '0' || line[pos] > '9' {
			return line
		}
	}
	return line[20:]
}

func releaseTagsFor(goVersion string) []string {
	major, minor, ok := parseGoVersion(goVersion)
	if !ok || major != 1 || minor < 1 {
		return nil
	}
	tags := make([]string, 0, minor)
	for i := 1; i <= minor; i++ {
		tags = append(tags, fmt.Sprintf("go1.%d", i))
	}
	return tags
}

func parseGoVersion(goVersion string) (int, int, bool) {
	if !strings.HasPrefix(goVersion, "go") {
		return 0, 0, false
	}
	body := strings.TrimPrefix(goVersion, "go")
	parts := strings.SplitN(body, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minorPart := parts[1]
	var digits strings.Builder
	for _, r := range minorPart {
		if r < '0' || r > '9' {
			break
		}
		digits.WriteRune(r)
	}
	if digits.Len() == 0 {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}

func (cfg xfailConfig) Match(goVersion, platform string, tc testCase) (bool, string) {
	return matchEntries(cfg.Entries, goVersion, platform, tc)
}

func (cfg xfailConfig) MatchFlaky(goVersion, platform string, tc testCase) (bool, string) {
	return matchEntries(cfg.Flakes, goVersion, platform, tc)
}

func (cfg xfailConfig) MatchHostSkip(goVersion, platform string, tc testCase) (bool, string) {
	return matchEntries(cfg.HostSkips, goVersion, platform, tc)
}

func (cfg xfailConfig) MatchTimeout(goVersion, platform string, tc testCase) (time.Duration, string, bool) {
	for _, entry := range cfg.Timeouts {
		if !entry.matches(goVersion, platform, tc) {
			continue
		}
		timeout, err := time.ParseDuration(entry.Timeout)
		if err != nil {
			return 0, fmt.Sprintf("invalid timeout override %q for %s: %v", entry.Timeout, entry.Case, err), false
		}
		reason := entry.Reason
		if reason == "" {
			reason = entry.Case
		}
		return timeout, reason, true
	}
	return 0, "", false
}

func matchEntries(entries []xfailEntry, goVersion, platform string, tc testCase) (bool, string) {
	for _, entry := range entries {
		if !entry.matches(goVersion, platform, tc) {
			continue
		}
		reason := entry.Reason
		if reason == "" {
			reason = entry.Case
		}
		return true, reason
	}
	return false, ""
}

func (entry xfailEntry) matches(goVersion, platform string, tc testCase) bool {
	return matchEntry(entry.Version, entry.Platform, entry.Directive, entry.Case, goVersion, platform, tc)
}

func (entry timeoutEntry) matches(goVersion, platform string, tc testCase) bool {
	return matchEntry(entry.Version, entry.Platform, entry.Directive, entry.Case, goVersion, platform, tc)
}

func matchEntry(version, platform, directive, casePattern, goVersion, goPlatform string, tc testCase) bool {
	if version != "" && !matchGoVersion(version, goVersion) {
		return false
	}
	if platform != "" && platform != goPlatform {
		return false
	}
	if directive != "" && directive != tc.Directive {
		return false
	}
	if casePattern == "" {
		return true
	}
	ok, err := path.Match(casePattern, tc.RelPath)
	return err == nil && ok
}

func matchGoVersion(version, goVersion string) bool {
	if goVersion == version {
		return true
	}
	suffix, ok := strings.CutPrefix(goVersion, version)
	if !ok {
		return false
	}
	return strings.HasPrefix(suffix, ".") || strings.HasPrefix(suffix, "rc") || strings.HasPrefix(suffix, "beta")
}

func splitQuoted(s string) ([]string, error) {
	var args []string
	arg := make([]rune, len(s))
	escaped := false
	quoted := false
	quote := '\x00'
	i := 0
	for _, r := range s {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
			continue
		case quote != '\x00':
			if r == quote {
				quote = '\x00'
				continue
			}
		case r == '"' || r == '\'':
			quoted = true
			quote = r
			continue
		case unicode.IsSpace(r):
			if quoted || i > 0 {
				quoted = false
				args = append(args, string(arg[:i]))
				i = 0
			}
			continue
		}
		arg[i] = r
		i++
	}
	if quoted || i > 0 {
		args = append(args, string(arg[:i]))
	}
	if quote != '\x00' {
		return args, errors.New("unclosed quote")
	}
	return args, nil
}

func parseDirectiveOptions(directive string, args []string, defaultRunTimeout time.Duration) (directiveOptions, error) {
	opts := directiveOptions{
		Timeout:   defaultRunTimeout,
		WantError: directive == "errorcheck",
	}
	args = append([]string(nil), args...)
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-1":
			opts.WantError = true
			args = args[1:]
		case "-0":
			opts.WantError = false
			args = args[1:]
		case "-s":
			opts.SingleFilePkgs = true
			args = args[1:]
		case "-t":
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for -t", directive)
			}
			secs, err := strconv.Atoi(args[1])
			if err != nil {
				return directiveOptions{}, fmt.Errorf("%s: invalid -t value %q: %w", directive, args[1], err)
			}
			opts.Timeout = time.Duration(secs) * time.Second
			args = args[2:]
		case "-goexperiment":
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for -goexperiment", directive)
			}
			opts.ExtraEnv = appendDirectiveEnv(opts.ExtraEnv, "GOEXPERIMENT", args[1])
			args = args[2:]
		case "-godebug":
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for -godebug", directive)
			}
			opts.ExtraEnv = appendDirectiveEnv(opts.ExtraEnv, "GODEBUG", args[1])
			args = args[2:]
		case "-gomodversion":
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for -gomodversion", directive)
			}
			opts.GoModVersion = args[1]
			args = args[2:]
		case "-gcflags", "-tags":
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for %s", directive, args[0])
			}
			opts.BuildFlags = append(opts.BuildFlags, args[0], args[1])
			args = args[2:]
		case "-ldflags":
			if usesCompilerFlags(directive) {
				opts.LinkerFlags = append(opts.LinkerFlags, args[1:]...)
				args = nil
				continue
			}
			if len(args) < 2 {
				return directiveOptions{}, fmt.Errorf("%s: missing value for -ldflags", directive)
			}
			payload := []string{args[1]}
			args = args[2:]
			for len(args) > 0 && strings.HasPrefix(args[0], "-") {
				switch args[0] {
				case "-1", "-0", "-s", "-t", "-goexperiment", "-godebug", "-gomodversion", "-gcflags", "-tags", "-ldflags":
					goto doneLDFlags
				}
				payload = append(payload, args[0])
				args = args[1:]
			}
		doneLDFlags:
			opts.BuildFlags = append(opts.BuildFlags, "-ldflags", strings.Join(payload, " "))
		default:
			if usesCompilerFlags(directive) {
				opts.CompilerFlags = append(opts.CompilerFlags, args[0])
			} else {
				opts.BuildFlags = append(opts.BuildFlags, args[0])
			}
			args = args[1:]
		}
	}
	opts.ProgramArgs = append(opts.ProgramArgs, args...)
	return opts, nil
}

func usesCompilerFlags(directive string) bool {
	switch directive {
	case "compile", "errorcheck", "errorcheckandrundir", "rundir":
		return true
	default:
		return false
	}
}

func appendDirectiveEnv(env []string, key, value string) []string {
	kv := key + "=" + value
	for i, item := range env {
		if strings.HasPrefix(item, key+"=") {
			env[i] = item + "," + value
			return env
		}
	}
	return append(env, kv)
}

func runCompileCase(t *testing.T, repoRoot, goroot, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	stdout, stderr, exitCode, elapsed, err := runLLGOCompiler(repoRoot, goroot, llgoBin, tc, opts, false, buildTimeout)
	if err != nil {
		return commandFailure("llgo tool compile", elapsed, err, stdout, stderr, exitCode)
	}
	if exitCode != 0 {
		return commandFailure("llgo tool compile", elapsed, fmt.Errorf("compiler exited unsuccessfully"), stdout, stderr, exitCode)
	}
	return nil
}

func runErrorCheckCase(t *testing.T, repoRoot, goroot, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	stdout, stderr, exitCode, elapsed, err := runLLGOCompiler(repoRoot, goroot, llgoBin, tc, opts, true, buildTimeout)
	if err != nil {
		return commandFailure("llgo tool compile", elapsed, err, stdout, stderr, exitCode)
	}
	if opts.WantError && exitCode == 0 {
		return fmt.Errorf("llgo tool compile succeeded unexpectedly")
	}
	if !opts.WantError && exitCode != 0 {
		return commandFailure("llgo tool compile", elapsed, fmt.Errorf("compiler exited unsuccessfully"), stdout, stderr, exitCode)
	}
	output := append(append([]byte(nil), stdout...), stderr...)
	output = filterNoise(normalizeOutput(output))
	return checkExpectedErrors(string(output), tc.SourcePath(), tc.FileName)
}

func (tc testCase) SourcePath() string {
	return filepath.Join(tc.Dir, tc.FileName)
}

func runLLGOCompiler(repoRoot, goroot, llgoBin string, tc testCase, opts directiveOptions, errorCheck bool, timeout time.Duration) ([]byte, []byte, int, time.Duration, error) {
	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	importCfg := filepath.Join(ws.rootDir, "importcfg")
	if err := os.WriteFile(importCfg, nil, 0o644); err != nil {
		return nil, nil, 0, 0, err
	}
	sourcePath := tc.SourcePath()
	if strings.HasSuffix(filepath.Base(tc.Dir), ".dir") {
		sourcePath, err = stageNestedCompilerCase(ws.workDir, tc)
		if err != nil {
			return nil, nil, 0, 0, err
		}
	}
	args := []string{"tool", "compile", "-e", "-p=p", "-importcfg=" + importCfg}
	if errorCheck {
		args = append(args, "-C", "-d=panic", "-o", filepath.Join(ws.rootDir, "a.o"))
	}
	args = append(args, opts.CompilerFlags...)
	if errorCheck && !hasSSACheckFlag(opts.CompilerFlags) {
		args = append(args, "-d=ssa/check/on")
	}
	args = append(args, sourcePath)
	env := runnerEnv(repoRoot, goroot, ws.gopath, opts.ExtraEnv)
	return runProgram(ws.workDir, llgoBin, env, timeout, args...)
}

func stageNestedCompilerCase(workDir string, tc testCase) (string, error) {
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(tc.Dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" {
			continue
		}
		src := filepath.Join(tc.Dir, name)
		targetDir := workDir
		if name != tc.FileName {
			targetDir = filepath.Join(workDir, strings.TrimSuffix(name, ".go"))
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return "", err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(targetDir, name), data, 0o644); err != nil {
			return "", err
		}
	}
	return filepath.Join(workDir, tc.FileName), nil
}

func hasSSACheckFlag(flags []string) bool {
	for _, compilerFlag := range flags {
		if strings.HasPrefix(compilerFlag, "-d=") && strings.Contains(compilerFlag, "ssa/check/") {
			return true
		}
	}
	return false
}

func runErrorCheckAndRunCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	srcDir := caseSourceDir(tc)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	var goFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && filepath.Ext(entry.Name()) == ".go" {
			goFiles = append(goFiles, entry.Name())
		}
	}
	sort.Strings(goFiles)
	pkgs, err := groupDirPackages(srcDir, goFiles, opts.SingleFilePkgs)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("%s: no Go packages in %s", tc.RelPath, srcDir)
	}

	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	if err := stageRundirLayout(ws.workDir, srcDir, opts.SingleFilePkgs); err != nil {
		return err
	}
	modVersion, err := toolchainGoModVersion(goroot)
	if err != nil {
		return err
	}
	if err := ensureModuleWorkspace(ws.workDir, "test", modVersion); err != nil {
		return err
	}
	importCfg := filepath.Join(ws.rootDir, "importcfg")
	if err := os.WriteFile(importCfg, nil, 0o644); err != nil {
		return err
	}
	env := runnerEnv(repoRoot, goroot, ws.gopath, append(opts.ExtraEnv, "GO111MODULE=on"))
	var diagnosticErrs []error
	for index, pkg := range pkgs {
		pkgDir := ws.workDir
		if pkg.name != "main" {
			pkgDir = filepath.Join(ws.workDir, pkg.dir)
		}
		args := []string{"tool", "compile", "-e", "-C", "-d=panic", "-importcfg=" + importCfg}
		args = append(args, opts.CompilerFlags...)
		var sources []diagnosticSource
		for _, fileName := range pkg.files {
			fullPath := filepath.Join(pkgDir, fileName)
			args = append(args, fullPath)
			sources = append(sources, diagnosticSource{full: fullPath, short: fileName})
		}
		stdout, stderr, exitCode, elapsed, runErr := runProgram(pkgDir, llgoBin, env, buildTimeout, args...)
		expectFailure := opts.WantError && index == len(pkgs)-2
		if runErr != nil {
			diagnosticErrs = append(diagnosticErrs, commandFailure("llgo tool compile", elapsed, runErr, stdout, stderr, exitCode))
			continue
		}
		if expectFailure && exitCode == 0 {
			diagnosticErrs = append(diagnosticErrs, fmt.Errorf("%s: package %s compiled successfully, want failure", tc.RelPath, pkg.name))
		}
		if !expectFailure && exitCode != 0 {
			diagnosticErrs = append(diagnosticErrs, commandFailure("llgo tool compile", elapsed, fmt.Errorf("compiler exited unsuccessfully"), stdout, stderr, exitCode))
		}
		output := append(append([]byte(nil), stdout...), stderr...)
		output = filterNoise(normalizeOutput(output))
		if checkErr := checkExpectedErrorsForFiles(string(output), sources); checkErr != nil {
			diagnosticErrs = append(diagnosticErrs, fmt.Errorf("%s: package %s diagnostics: %w", tc.RelPath, pkg.name, checkErr))
		}
	}

	runErr := runBuildAndCompare(t, tc.RelPath, ws.workDir, ws.rootDir, env, goCmd, llgoBin, nil, nil, opts.ProgramArgs, buildTimeout, opts.Timeout)
	if runErr != nil {
		diagnosticErrs = append(diagnosticErrs, runErr)
	}
	return errors.Join(diagnosticErrs...)
}

func runSingleFileCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	goBin := filepath.Join(ws.rootDir, "go.out")
	llgoOut := filepath.Join(ws.rootDir, "llgo.out")
	metrics := caseMetrics{}
	sourceFiles, programArgs := splitSourceFiles(tc.FileName, opts.ProgramArgs)
	buildTarget := tc.FileName
	nestedDirCase := strings.HasSuffix(filepath.Base(tc.Dir), ".dir")
	extraEnv := append([]string{}, opts.ExtraEnv...)
	if nestedDirCase {
		preserveLayout, err := directoryHasSubdirectories(tc.Dir)
		if err != nil {
			return err
		}
		modulePath := "test"
		if preserveLayout {
			if err := overlayDir(ws.workDir, tc.Dir); err != nil {
				return err
			}
			modulePath = filepath.Base(tc.Dir)
		} else if err := stageRundirLayout(ws.workDir, tc.Dir, false); err != nil {
			return err
		}
		modVersion, err := toolchainGoModVersion(goroot)
		if err != nil {
			return err
		}
		if err := ensureModuleWorkspace(ws.workDir, modulePath, modVersion); err != nil {
			return err
		}
		extraEnv = append(extraEnv, "GO111MODULE=on")
		buildTarget = "."
	} else if len(sourceFiles) > 1 {
		if err := stageSelectedFiles(ws.workDir, tc.Dir, sourceFiles); err != nil {
			return err
		}
		buildTarget = "."
	} else {
		if err := overlayDir(ws.workDir, tc.Dir); err != nil {
			return err
		}
	}
	env := runnerEnv(repoRoot, goroot, ws.gopath, extraEnv)

	goBuildStdout, goBuildStderr, goBuildExit, goBuildDur, err := runProgram(ws.workDir, goCmd, env, buildTimeout, append([]string{"build"}, append(opts.BuildFlags, "-o", goBin, buildTarget)...)...)
	metrics.goBuild += goBuildDur
	if cmdErr := requireSuccessfulExit(err, goBuildExit); cmdErr != nil {
		return commandFailure("baseline go build", goBuildDur, cmdErr, goBuildStdout, goBuildStderr, goBuildExit)
	}
	if err := ensureBuiltBinary(goBin, "baseline go build"); err != nil {
		return err
	}
	llgoBuildStdout, llgoBuildStderr, llgoBuildExit, llgoBuildDur, err := runProgram(ws.workDir, llgoBin, env, buildTimeout, append([]string{"build"}, append(opts.BuildFlags, "-o", llgoOut, buildTarget)...)...)
	metrics.llgoBuild += llgoBuildDur
	if cmdErr := requireSuccessfulExit(err, llgoBuildExit); cmdErr != nil {
		return commandFailure("llgo build", llgoBuildDur, cmdErr, llgoBuildStdout, llgoBuildStderr, llgoBuildExit)
	}
	if err := ensureBuiltBinary(llgoOut, "llgo build"); err != nil {
		return err
	}

	runDir := ws.workDir
	if tc.Directive == "run" {
		runDir = filepath.Join(goroot, "test")
	}
	runEnv := restoreProcessEnv(append([]string{}, env...), "GO111MODULE")
	runEnv = restoreProcessEnv(runEnv, "GOPATH")
	goStdout, goStderr, goExit, goRunDur, err := runProgram(runDir, goBin, runEnv, opts.Timeout, programArgs...)
	metrics.goRun += goRunDur
	if cmdErr := requireSuccessfulExit(err, goExit); cmdErr != nil {
		return commandFailure("baseline go run", goRunDur, cmdErr, goStdout, goStderr, goExit)
	}
	llgoStdout, llgoStderr, llgoExit, llgoRunDur, err := runProgram(runDir, llgoOut, runEnv, opts.Timeout, programArgs...)
	metrics.llgoRun += llgoRunDur
	if cmdErr := requireSuccessfulExit(err, llgoExit); cmdErr != nil {
		return commandFailure("llgo run", llgoRunDur, cmdErr, llgoStdout, llgoStderr, llgoExit)
	}

	logSlowCase(t, tc.RelPath, metrics.goBuild, metrics.llgoBuild, metrics.goRun, metrics.llgoRun)
	return compareOutputs(goStdout, goStderr, goExit, llgoStdout, llgoStderr, llgoExit)
}

func runOutputCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	sourceFiles, programArgs := splitSourceFiles(tc.FileName, opts.ProgramArgs)
	env := runnerEnv(repoRoot, goroot, ws.gopath, opts.ExtraEnv)
	metrics := caseMetrics{}

	genWS, err := stageRunOutputWorkspace(ws, "go", tc.Dir, sourceFiles)
	if err != nil {
		return err
	}

	goModVersion, err := toolchainGoModVersion(goroot)
	if err != nil {
		return err
	}

	goGen, goGenRun, err := generateRunOutput(genWS, goCmd, env, sourceFiles, programArgs, opts.Timeout, false, "go", goModVersion)
	metrics.goRun += goGenRun
	if err != nil {
		return err
	}

	goWS, err := stageRunOutputWorkspace(ws, "go-generated", genWS.workDir, []string{goGen})
	if err != nil {
		return err
	}
	llgoWS, err := stageRunOutputWorkspace(ws, "llgo-generated", genWS.workDir, []string{goGen})
	if err != nil {
		return err
	}

	goStdout, goStderr, goExit, goBuildDur, goRunDur, err := runGeneratedProgram(goWS, goCmd, env, goGen, "go", buildTimeout, opts.Timeout)
	metrics.goBuild += goBuildDur
	metrics.goRun += goRunDur
	if err != nil {
		return err
	}
	llgoStdout, llgoStderr, llgoExit, llgoBuildDur, llgoRunDur, err := runGeneratedProgram(llgoWS, llgoBin, env, goGen, "llgo", buildTimeout, opts.Timeout)
	metrics.llgoBuild += llgoBuildDur
	metrics.llgoRun += llgoRunDur
	if err != nil {
		return err
	}

	logSlowCase(t, tc.RelPath, metrics.goBuild, metrics.llgoBuild, metrics.goRun, metrics.llgoRun)
	return compareOutputs(goStdout, goStderr, goExit, llgoStdout, llgoStderr, llgoExit)
}

func stageRunOutputWorkspace(base caseWorkspace, label, srcDir string, sourceFiles []string) (caseWorkspace, error) {
	workDir := filepath.Join(base.rootDir, label+"-work")
	if err := stageSelectedFiles(workDir, srcDir, sourceFiles); err != nil {
		return caseWorkspace{}, err
	}
	toolWS := base
	toolWS.workDir = workDir
	return toolWS, nil
}

func splitSourceFiles(entry string, args []string) ([]string, []string) {
	files := []string{entry}
	i := 0
	for i < len(args) {
		arg := filepath.Clean(args[i])
		if strings.HasSuffix(arg, ".go") && !filepath.IsAbs(arg) {
			files = append(files, arg)
			i++
			continue
		}
		break
	}
	return files, args[i:]
}

func runInDirCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, opts directiveOptions, buildTimeout time.Duration) error {
	t.Helper()
	srcDir := caseSourceDir(tc)
	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	if err := overlayDir(ws.workDir, srcDir); err != nil {
		return err
	}
	modVersion := opts.GoModVersion
	if modVersion == "" {
		modVersion = "1.14"
	}
	modName := filepath.Base(srcDir)
	if err := os.WriteFile(filepath.Join(ws.workDir, "go.mod"), []byte(fmt.Sprintf("module %s\ngo %s\n", modName, modVersion)), 0o644); err != nil {
		return err
	}
	env := runnerEnv(repoRoot, goroot, ws.gopath, append(opts.ExtraEnv, "GO111MODULE=on"))
	buildFlags := append([]string{}, opts.BuildFlags...)
	return runBuildAndCompare(t, tc.RelPath, ws.workDir, ws.rootDir, env, goCmd, llgoBin, buildFlags, buildFlags, opts.ProgramArgs, buildTimeout, opts.Timeout)
}

func runDirCase(t *testing.T, repoRoot, goroot, goCmd, llgoBin string, tc testCase, opts directiveOptions, preserveLayout bool, buildTimeout time.Duration) error {
	t.Helper()
	srcDir := caseSourceDir(tc)
	ws, err := prepareCaseWorkspace(repoRoot)
	if err != nil {
		return err
	}
	if !*flagKeep {
		defer ws.cleanup()
	}
	if preserveLayout {
		if err := overlayDir(ws.workDir, srcDir); err != nil {
			return err
		}
	} else {
		if err := stageRundirLayout(ws.workDir, srcDir, opts.SingleFilePkgs); err != nil {
			return err
		}
	}
	modVersion, err := toolchainGoModVersion(goroot)
	if err != nil {
		return err
	}
	if err := ensureModuleWorkspace(ws.workDir, "test", modVersion); err != nil {
		return err
	}
	extraEnv := append([]string{}, opts.ExtraEnv...)
	extraEnv = append(extraEnv, "GO111MODULE=on")
	env := runnerEnv(repoRoot, goroot, ws.gopath, extraEnv)
	goBuildFlags, llgoBuildFlags := directoryBuildFlags(opts)
	return runBuildAndCompare(t, tc.RelPath, ws.workDir, ws.rootDir, env, goCmd, llgoBin, goBuildFlags, llgoBuildFlags, opts.ProgramArgs, buildTimeout, opts.Timeout)
}

func directoryBuildFlags(opts directiveOptions) (goFlags, llgoFlags []string) {
	goFlags = append(goFlags, opts.BuildFlags...)
	llgoFlags = append(llgoFlags, opts.BuildFlags...)
	if len(opts.CompilerFlags) != 0 {
		value := strings.Join(opts.CompilerFlags, " ")
		goFlags = append(goFlags, "-gcflags="+value)
		llgoFlags = append(llgoFlags, "-gcflags="+value)
	}
	if len(opts.LinkerFlags) != 0 {
		value := strings.Join(opts.LinkerFlags, " ")
		goFlags = append(goFlags, "-ldflags="+value)
		llgoFlags = append(llgoFlags, "-ldflags="+value)
	}
	return goFlags, llgoFlags
}

func runBuildAndCompare(t *testing.T, casePath, workDir, rootDir string, env []string, goCmd, llgoBin string, goBuildFlags, llgoBuildFlags, programArgs []string, buildTimeout, runTimeout time.Duration) error {
	t.Helper()
	goBin := filepath.Join(rootDir, "go.out")
	llgoOut := filepath.Join(rootDir, "llgo.out")
	metrics := caseMetrics{}

	goBuildStdout, goBuildStderr, goBuildExit, goBuildDur, err := runProgram(workDir, goCmd, env, buildTimeout, append([]string{"build"}, append(goBuildFlags, "-o", goBin, ".")...)...)
	metrics.goBuild += goBuildDur
	if cmdErr := requireSuccessfulExit(err, goBuildExit); cmdErr != nil {
		return commandFailure("baseline go build", goBuildDur, cmdErr, goBuildStdout, goBuildStderr, goBuildExit)
	}
	if err := ensureBuiltBinary(goBin, "baseline go build"); err != nil {
		return err
	}
	llgoBuildStdout, llgoBuildStderr, llgoBuildExit, llgoBuildDur, err := runProgram(workDir, llgoBin, env, buildTimeout, append([]string{"build"}, append(llgoBuildFlags, "-o", llgoOut, ".")...)...)
	metrics.llgoBuild += llgoBuildDur
	if cmdErr := requireSuccessfulExit(err, llgoBuildExit); cmdErr != nil {
		return commandFailure("llgo build", llgoBuildDur, cmdErr, llgoBuildStdout, llgoBuildStderr, llgoBuildExit)
	}
	if err := ensureBuiltBinary(llgoOut, "llgo build"); err != nil {
		return err
	}

	goStdout, goStderr, goExit, goRunDur, err := runProgram(workDir, goBin, env, runTimeout, programArgs...)
	metrics.goRun += goRunDur
	if cmdErr := requireSuccessfulExit(err, goExit); cmdErr != nil {
		return commandFailure("baseline go run", goRunDur, cmdErr, goStdout, goStderr, goExit)
	}
	llgoStdout, llgoStderr, llgoExit, llgoRunDur, err := runProgram(workDir, llgoOut, env, runTimeout, programArgs...)
	metrics.llgoRun += llgoRunDur
	if cmdErr := requireSuccessfulExit(err, llgoExit); cmdErr != nil {
		return commandFailure("llgo run", llgoRunDur, cmdErr, llgoStdout, llgoStderr, llgoExit)
	}

	logSlowCase(t, casePath, metrics.goBuild, metrics.llgoBuild, metrics.goRun, metrics.llgoRun)
	return compareOutputs(goStdout, goStderr, goExit, llgoStdout, llgoStderr, llgoExit)
}

func compareOutputs(goStdout, goStderr []byte, goExit int, llgoStdout, llgoStderr []byte, llgoExit int) error {
	goStdout = normalizeOutput(goStdout)
	goStderr = normalizeOutput(goStderr)
	llgoStdout = normalizeOutput(filterNoise(llgoStdout))
	llgoStderr = normalizeOutput(filterNoise(llgoStderr))
	if !bytes.Equal(llgoStdout, goStdout) {
		return fmt.Errorf("stdout mismatch\nllgo:\n%s\n\ngo:\n%s", llgoStdout, goStdout)
	}
	if !bytes.Equal(llgoStderr, goStderr) {
		return fmt.Errorf("stderr mismatch\nllgo:\n%s\n\ngo:\n%s", llgoStderr, goStderr)
	}
	if llgoExit != goExit {
		return fmt.Errorf("exit code mismatch: llgo=%d go=%d", llgoExit, goExit)
	}
	return nil
}

func directoryHasSubdirectories(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			return true, nil
		}
	}
	return false, nil
}

func caseSourceDir(tc testCase) string {
	base := strings.TrimSuffix(tc.FileName, filepath.Ext(tc.FileName)) + ".dir"
	return filepath.Join(tc.Dir, base)
}

func generateRunOutput(ws caseWorkspace, tool string, env []string, sourceFiles, programArgs []string, timeout time.Duration, packageRun bool, label, goModVersion string) (string, time.Duration, error) {
	if packageRun {
		if err := ensureModuleWorkspace(ws.workDir, "llgo-goroot-runoutput", goModVersion); err != nil {
			return "", 0, err
		}
		env = upsertEnv(env, "GO111MODULE=on")
	}
	args := []string{"run"}
	if packageRun {
		args = append(args, ".")
	} else {
		args = append(args, sourceFiles...)
	}
	args = append(args, programArgs...)
	stdout, stderr, exitCode, runDur, err := runProgram(
		ws.workDir,
		tool,
		env,
		timeout,
		args...,
	)
	if cmdErr := requireSuccessfulExit(err, exitCode); cmdErr != nil {
		return "", runDur, commandFailure(label+" generate run", runDur, cmdErr, stdout, stderr, exitCode)
	}
	genBase := label + "_tmp__.go"
	genFile := filepath.Join(ws.workDir, genBase)
	if err := os.WriteFile(genFile, stdout, 0o644); err != nil {
		return "", runDur, err
	}
	return genBase, runDur, nil
}

func ensureModuleWorkspace(dir, modulePath, goVersion string) error {
	modFile := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(modFile); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	data := fmt.Sprintf("module %s\ngo %s\n", modulePath, goVersion)
	return os.WriteFile(modFile, []byte(data), 0o644)
}

func toolchainGoModVersion(goroot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read %s/VERSION: %w", goroot, err)
	}
	version := strings.TrimSpace(string(data))
	version = strings.TrimPrefix(version, "devel ")
	version = strings.TrimPrefix(version, "go")
	if version == "" {
		return "", fmt.Errorf("parse %s/VERSION: empty version", goroot)
	}
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("parse %s/VERSION: unexpected version %q", goroot, string(data))
	}
	return parts[0] + "." + parts[1], nil
}

func runGeneratedProgram(ws caseWorkspace, tool string, env []string, fileName, label string, buildTimeout, runTimeout time.Duration) ([]byte, []byte, int, time.Duration, time.Duration, error) {
	out := filepath.Join(ws.rootDir, label+"-generated.out")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	buildStdout, buildStderr, buildExit, buildDur, err := runProgram(ws.workDir, tool, env, buildTimeout, "build", "-o", out, fileName)
	if cmdErr := requireSuccessfulExit(err, buildExit); cmdErr != nil {
		return nil, nil, 0, buildDur, 0, commandFailure(label+" generated build", buildDur, cmdErr, buildStdout, buildStderr, buildExit)
	}
	if err := ensureBuiltBinary(out, label+" generated build"); err != nil {
		return nil, nil, 0, buildDur, 0, err
	}
	stdout, stderr, exitCode, runDur, err := runProgram(ws.workDir, out, env, runTimeout)
	if cmdErr := requireSuccessfulExit(err, exitCode); cmdErr != nil {
		return nil, nil, 0, buildDur, runDur, commandFailure(label+" generated run", runDur, cmdErr, stdout, stderr, exitCode)
	}
	return stdout, stderr, exitCode, buildDur, runDur, nil
}

func overlayDir(dstRoot, srcRoot string) error {
	dstRoot = filepath.Clean(dstRoot)
	if err := os.MkdirAll(dstRoot, 0o777); err != nil {
		return err
	}
	srcRoot, err := filepath.Abs(srcRoot)
	if err != nil {
		return err
	}
	return filepath.WalkDir(srcRoot, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil || srcPath == srcRoot {
			return err
		}
		suffix := strings.TrimPrefix(srcPath, srcRoot)
		for len(suffix) > 0 && suffix[0] == filepath.Separator {
			suffix = suffix[1:]
		}
		dstPath := filepath.Join(dstRoot, suffix)
		var info fs.FileInfo
		if d.Type()&os.ModeSymlink != 0 {
			info, err = os.Stat(srcPath)
		} else {
			info, err = d.Info()
		}
		if err != nil {
			return err
		}
		perm := info.Mode() & os.ModePerm
		if info.IsDir() {
			return os.MkdirAll(dstPath, perm|0o200)
		}
		if err := os.Symlink(srcPath, dstPath); err == nil {
			return nil
		}
		src, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
		return err
	})
}

func stageSelectedFiles(dstRoot, srcRoot string, files []string) error {
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	for _, name := range files {
		clean := filepath.Clean(name)
		if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsupported relative source path %q", name)
		}
		srcPath := filepath.Join(srcRoot, clean)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstRoot, clean)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type dirPackage struct {
	name  string
	files []string
	dir   string
}

func stageRundirLayout(dstRoot, srcRoot string, singleFilePkgs bool) error {
	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		return err
	}
	var goFiles []string
	var auxFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		switch filepath.Ext(name) {
		case ".go":
			goFiles = append(goFiles, name)
		case ".s":
			auxFiles = append(auxFiles, name)
		}
	}
	sort.Strings(goFiles)
	sort.Strings(auxFiles)
	pkgs, err := groupDirPackages(srcRoot, goFiles, singleFilePkgs)
	if err != nil {
		return err
	}
	hasMissingBody := false
	for _, pkg := range pkgs {
		targetDir := dstRoot
		if pkg.name != "main" {
			targetDir = filepath.Join(dstRoot, pkg.dir)
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		for _, fileName := range pkg.files {
			data, err := os.ReadFile(filepath.Join(srcRoot, fileName))
			if err != nil {
				return err
			}
			data, err = rewriteRelativeImports(data, "test")
			if err != nil {
				return err
			}
			missingBody, err := sourceHasMissingFunctionBody(data)
			if err != nil {
				return err
			}
			hasMissingBody = hasMissingBody || missingBody
			if err := os.WriteFile(filepath.Join(targetDir, stagedRundirFileName(fileName)), data, 0o644); err != nil {
				return err
			}
		}
	}
	for _, name := range auxFiles {
		data, err := os.ReadFile(filepath.Join(srcRoot, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstRoot, name), data, 0o644); err != nil {
			return err
		}
	}
	if hasMissingBody && len(auxFiles) == 0 {
		if err := os.WriteFile(filepath.Join(dstRoot, "testdir_empty.s"), nil, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func stagedRundirFileName(name string) string {
	if strings.HasSuffix(name, "_test.go") {
		return strings.TrimSuffix(name, "_test.go") + "_testdir.go"
	}
	return name
}

func sourceHasMissingFunctionBody(src []byte) (bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return false, err
	}
	missing := false
	ast.Inspect(file, func(node ast.Node) bool {
		if decl, ok := node.(*ast.FuncDecl); ok && decl.Body == nil {
			missing = true
			return false
		}
		return !missing
	})
	return missing, nil
}

func groupDirPackages(srcRoot string, goFiles []string, singleFilePkgs bool) ([]dirPackage, error) {
	var pkgs []dirPackage
	indexByName := map[string]int{}
	for _, name := range goFiles {
		pkgName, err := getPackageNameFromSource(filepath.Join(srcRoot, name))
		if err != nil {
			return nil, err
		}
		if singleFilePkgs {
			pkgs = append(pkgs, dirPackage{name: pkgName, files: []string{name}, dir: strings.TrimSuffix(name, ".go")})
			continue
		}
		if i, ok := indexByName[pkgName]; ok {
			pkgs[i].files = append(pkgs[i].files, name)
			continue
		}
		pkgs = append(pkgs, dirPackage{name: pkgName, files: []string{name}, dir: strings.TrimSuffix(name, ".go")})
		indexByName[pkgName] = len(pkgs) - 1
	}
	return pkgs, nil
}

func getPackageNameFromSource(filePath string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	return file.Name.Name, nil
}

func rewriteRelativeImports(src []byte, modulePath string) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	changed := false
	for _, imp := range file.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(p, "./") {
			imp.Path.Value = strconv.Quote(path.Join(modulePath, strings.TrimPrefix(p, "./")))
			changed = true
		}
	}
	if !changed {
		return src, nil
	}
	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func ensureBuiltBinary(path, step string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s succeeded but did not produce %s", step, path)
		}
		return fmt.Errorf("%s output %s: %w", step, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s produced directory %s, want executable file", step, path)
	}
	return nil
}
