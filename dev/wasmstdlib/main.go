// Command wasmstdlib runs a bounded standard-library acceptance slice and
// records the rest of the source inventory as unvalidated, never as passing.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const packagePrefix = "github.com/xgo-dev/llgo/test/std/"

type testCase struct{ Package, Witness string }

var acceptance = []testCase{
	{"errors", "TestAs"},
	{"sort", "TestSliceHelpers"},
	{"encoding/binary", "TestReadWriteStruct"},
	{"fmt", "TestFormatterInterface"},
	{"strconv", "TestIntSizeMatchesUintSize"},
	{"io", "TestPipeWriterAndReaderCloseWithError"},
}

type profile struct {
	Name, GOOS, Target string
	Reference          bool
}

func selectProfile(name string) (profile, error) {
	switch name {
	case "J32-Emscripten":
		return profile{name, "js", "emscripten", false}, nil
	case "J64-Emscripten":
		return profile{name, "js", "emscripten-memory64", false}, nil
	case "J32-GoJS":
		return profile{Name: name, GOOS: "js"}, nil
	case "W32-WASI":
		return profile{name, "wasip1", "wasi", false}, nil
	case "GoJS-reference":
		return profile{Name: name, GOOS: "js", Reference: true}, nil
	case "GoWASI-reference":
		return profile{Name: name, GOOS: "wasip1", Reference: true}, nil
	}
	return profile{}, fmt.Errorf("unknown profile %q", name)
}

func sourceContext(p profile) (tags, cgo string) {
	if p.Reference {
		return "", "0"
	}
	tags, cgo = "llgo,osusergo,llgo.wasm.gc.linear", "0"
	switch p.Target {
	case "emscripten":
		return tags + ",llgo.wasm.emscripten", "1"
	case "emscripten-memory64":
		return tags + ",llgo.wasm.emscripten,llgo.wasm.emscripten.memory64", "1"
	case "wasi":
		return tags + ",llgo.wasm.wasi", "1"
	default:
		return tags, cgo
	}
}

type entry struct {
	Package        string `json:"package"`
	SourceSelected bool   `json:"source_selected"`
	SelectedForRun bool   `json:"selected_for_run"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	Tests          int    `json:"passed_top_level_tests"`
}

type report struct {
	Schema         int     `json:"schema"`
	Profile        string  `json:"profile"`
	Implementation string  `json:"implementation"`
	Contract       string  `json:"contract"`
	GoVersion      string  `json:"go_version"`
	Result         string  `json:"slice_result"`
	Reason         string  `json:"reason,omitempty"`
	Packages       []entry `json:"packages"`
}

// Inspect test directories independently of go list so native-only build tags
// cannot silently remove packages from the report's denominator.
func discover(root string) ([]string, error) {
	base := filepath.Join(root, "test", "std")
	seen := make(map[string]bool)
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != base && (d.Name() == "testdata" || d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			// WalkDir only returns descendants of base, so Rel cannot cross a
			// volume or otherwise fail here.
			rel, _ := filepath.Rel(base, filepath.Dir(path))
			seen[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, err
}

func sourceSelection(data []byte) (map[string]bool, error) {
	selected := make(map[string]bool)
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var pkg struct {
			ImportPath                string
			TestGoFiles, XTestGoFiles []string
			Error                     *struct{ Err string }
			DepsErrors                []struct{ Err string }
		}
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				return selected, nil
			}
			return nil, err
		}
		if pkg.Error != nil {
			return nil, fmt.Errorf("source selection %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		if len(pkg.DepsErrors) != 0 {
			return nil, fmt.Errorf("source selection %s: %s", pkg.ImportPath, pkg.DepsErrors[0].Err)
		}
		if !strings.HasPrefix(pkg.ImportPath, packagePrefix) {
			return nil, fmt.Errorf("unexpected source package %q", pkg.ImportPath)
		}
		if len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) != 0 {
			selected[strings.TrimPrefix(pkg.ImportPath, packagePrefix)] = true
		}
	}
}

func inventory(names []string, selected map[string]bool, cases []testCase) ([]entry, error) {
	if len(cases) == 0 {
		return nil, errors.New("acceptance slice must not be empty")
	}
	var entries []entry
	indices := make(map[string]int)
	for _, name := range names {
		e := entry{Package: name, SourceSelected: selected[name], Status: "not-run", Reason: "outside this acceptance slice; not validated"}
		if !e.SourceSelected {
			e.Status = "source-excluded"
			e.Reason = "no tests selected for this Go version/source context; requires explicit classification or replacement coverage"
		}
		indices[name] = len(entries)
		entries = append(entries, e)
	}
	seen := make(map[string]bool)
	for _, tc := range cases {
		i, exists := indices[tc.Package]
		if !exists || !selected[tc.Package] || seen[tc.Package] || tc.Witness == "" {
			return nil, fmt.Errorf("invalid or source-excluded acceptance package %q", tc.Package)
		}
		seen[tc.Package] = true
		entries[i].SelectedForRun = true
		entries[i].Reason = "not executed yet"
	}
	return entries, nil
}

func validateOutput(output []byte, witness string) (int, error) {
	passes, tests, found := 0, 0, false
	for _, raw := range strings.Split(string(output), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if line == "PASS" {
			passes++
		}
		// A test may write stdout without a trailing newline (fmt.Print is
		// tested this way). Its result record then follows user output on the
		// same line. Subtest names contain '/', regardless of indentation.
		marker := strings.LastIndex(line, "--- ")
		if marker < 0 {
			continue
		}
		status, rest, ok := strings.Cut(line[marker+4:], ": ")
		fields := strings.Fields(rest)
		validRecord := ok && len(fields) == 2 && strings.HasPrefix(fields[1], "(") && strings.HasSuffix(fields[1], ")")
		if status == "FAIL" || status == "SKIP" && (!validRecord || !strings.Contains(fields[0], "/")) {
			return 0, fmt.Errorf("failed or skipped test: %s", line[marker:])
		}
		if validRecord && status == "PASS" && !strings.Contains(fields[0], "/") {
			tests++
			found = found || fields[0] == witness
		}
	}
	if passes != 1 || !found {
		return 0, fmt.Errorf("incomplete execution: terminal PASS records=%d, witness %s=%v", passes, witness, found)
	}
	return tests, nil
}

type command struct {
	Program string
	Args    []string
	Env     map[string]string
}

func testCommand(p profile, goCmd, llgo, goRoot, pkg string) command {
	args := []string{"test", "-v", "-count=1", "-timeout=60s"}
	if p.Reference {
		helper := filepath.Join(goRoot, "lib", "wasm", "go_"+p.GOOS+"_wasm_exec")
		args = append(args, "-exec="+strconv.Quote(helper), "./test/std/"+pkg)
		return command{goCmd, args, map[string]string{"GOOS": p.GOOS, "GOARCH": "wasm", "CGO_ENABLED": "0", "GOWASIRUNTIME": "wasmtime", "GOMAXPROCS": "1"}}
	}
	env := map[string]string{"LLGO_BUILD_CACHE": "on"}
	if p.Target != "" {
		args = append(args, "-target", p.Target, "-emulator")
	} else {
		env["GOOS"], env["GOARCH"], env["CGO_ENABLED"] = p.GOOS, "wasm", "0"
	}
	args = append(args, "./test/std/"+pkg)
	return command{llgo, args, env}
}

func subprocess(root string, c command) *exec.Cmd {
	cmd := exec.Command(c.Program, c.Args...)
	cmd.Dir = root
	// Inherited GOFLAGS can filter tests or alter source tags, turning a full
	// package run into an undocumented subset even when its witness passes.
	// An empty GOFLAGS alone falls back to flags saved by go env -w, so also
	// isolate persistent Go configuration. Explicit process environment such
	// as GOPROXY remains available for dependency downloads.
	env := map[string]string{"LLGO_ROOT": root, "GOWORK": "off", "GOENV": "off", "GOFLAGS": ""}
	for k, v := range c.Env {
		env[k] = v
	}
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if _, replaced := env[key]; !replaced && key != "GOOS" && key != "GOARCH" {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd
}

func execute(root string, c command) ([]byte, error) {
	return subprocess(root, c).CombinedOutput()
}

func executeStructured(root string, c command) ([]byte, error) {
	cmd := subprocess(root, c)
	// Cold go list runs may print module downloads on stderr. Those messages
	// must remain visible without contaminating the JSON stdout stream.
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func runSlice(r *report, cases []testCase, run func(testCase) ([]byte, error), save func(*report) error) error {
	r.Result = "running"
	if err := save(r); err != nil {
		return err
	}
	for _, tc := range cases {
		var e *entry
		for i := range r.Packages {
			if r.Packages[i].Package == tc.Package {
				e = &r.Packages[i]
			}
		}
		if e == nil || !e.SelectedForRun {
			return fmt.Errorf("package %s is absent from acceptance inventory", tc.Package)
		}
		output, err := run(tc)
		if err == nil {
			e.Tests, err = validateOutput(output, tc.Witness)
		}
		e.Status, e.Reason = "pass", ""
		if err != nil {
			e.Status, e.Reason, e.Tests = "fail", err.Error(), 0
			r.Result = "fail"
		}
		if saveErr := save(r); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		if err != nil {
			return fmt.Errorf("%s/%s: %w", r.Profile, tc.Package, err)
		}
	}
	r.Result = "pass"
	return save(r)
}

var (
	exitProcess             = os.Exit
	processStderr io.Writer = os.Stderr
	getwdForRun             = os.Getwd
	writeRunFile            = os.WriteFile
)

func main() {
	exitProcess(runMain(os.Args[1:], processStderr))
}

func runMain(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("wasmstdlib", flag.ContinueOnError)
	flags.SetOutput(stderr)
	full := flags.Bool("full", false, "audit all test/ packages, continuing after failures")
	shard := flags.Int("shard", 0, "full-audit shard index")
	shards := flags.Int("shards", 1, "full-audit shard count")
	profileName := flags.String("profile", "", "J32-GoJS, J32-Emscripten, J64-Emscripten, W32-WASI, GoJS-reference, or GoWASI-reference")
	reportPath := flags.String("report", "", "output JSON file (required)")
	llgo := flags.String("llgo", "llgo", "LLGo executable")
	goCmd := flags.String("go", "go", "Go executable")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	var err error
	if *full {
		err = runFull(*profileName, *reportPath, *goCmd, *llgo, *shard, *shards)
	} else {
		err = run(*profileName, *reportPath, *goCmd, *llgo)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func run(name, reportPath, goCmd, llgo string) (retErr error) {
	if reportPath == "" {
		return errors.New("-report is required")
	}
	r := &report{Schema: 1, Profile: name, Result: "preparing", Packages: []entry{}}
	save := func(r *report) error {
		// report contains only JSON scalar values and slices, so this fixed
		// schema cannot produce an unsupported-value or cycle error.
		data, _ := json.MarshalIndent(r, "", "  ")
		return writeRunFile(reportPath, append(data, '\n'), 0o644)
	}
	// Every invocation replaces an earlier result before preparing commands.
	// Preserve errors from preparation and summary writing as well as tests.
	defer func() {
		if retErr != nil {
			r.Result, r.Reason = "fail", retErr.Error()
			if err := save(r); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("save failure report: %w", err))
			}
		}
	}()
	if err := save(r); err != nil {
		return fmt.Errorf("initialize report: %w", err)
	}
	p, err := selectProfile(name)
	if err != nil {
		return err
	}
	r.Implementation, r.Contract = "llgo", "LLGo profile behavior; not official-Go binary compatibility"
	if p.Reference {
		r.Implementation, r.Contract = "go-reference", "official Go compiler and host helper; not LLGo output"
	}
	root, err := getwdForRun()
	if err != nil {
		return err
	}
	names, err := discover(root)
	for _, name := range names {
		e := entry{Package: name, Status: "not-run", Reason: "source selection not completed; not validated"}
		for _, tc := range acceptance {
			if tc.Package == name {
				e.SelectedForRun = true
			}
		}
		r.Packages = append(r.Packages, e)
	}
	if err != nil {
		return fmt.Errorf("discover source inventory: %w", err)
	}
	if err := save(r); err != nil {
		return fmt.Errorf("save source inventory: %w", err)
	}
	goEnv, err := executeStructured(root, command{goCmd, []string{"env", "-json", "GOROOT", "GOVERSION"}, nil})
	if err != nil {
		return fmt.Errorf("go env: %w: %s", err, goEnv)
	}
	var env struct{ GOROOT, GOVERSION string }
	if err := json.Unmarshal(goEnv, &env); err != nil {
		return fmt.Errorf("decode go env: %w", err)
	}
	r.GoVersion = env.GOVERSION
	args := []string{"list", "-e", "-json"}
	tags, cgo := sourceContext(p)
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, "./test/std/...")
	data, err := executeStructured(root, command{goCmd, args, map[string]string{"GOOS": p.GOOS, "GOARCH": "wasm", "CGO_ENABLED": cgo}})
	if err != nil {
		return fmt.Errorf("go list: %w: %s", err, data)
	}
	selected, err := sourceSelection(data)
	if err != nil {
		return fmt.Errorf("go list source selection: %w", err)
	}
	entries, err := inventory(names, selected, acceptance)
	if err != nil {
		return err
	}
	r.Packages = entries
	err = runSlice(r, acceptance, func(tc testCase) ([]byte, error) {
		fmt.Printf("running %s/%s (%s)\n", name, tc.Package, r.Implementation)
		out, err := execute(root, testCommand(p, goCmd, llgo, env.GOROOT, tc.Package))
		fmt.Print(string(out))
		return out, err
	}, save)
	if err != nil {
		r.Result, r.Reason = "fail", err.Error()
	}
	counts := map[string]int{}
	for _, e := range r.Packages {
		counts[e.Status]++
	}
	summary := fmt.Sprintf("### W2 standard-library slice: %s\n\n%s\n\nGo version: %s; slice result: %s.\n\nPassed packages: %d; failed: %d; not run: %d; source-excluded (unclassified): %d.\n\nOnly this slice was checked; the complete inventory is a W3 gate.\n", name, r.Contract, r.GoVersion, r.Result, counts["pass"], counts["fail"], counts["not-run"], counts["source-excluded"])
	fmt.Print(summary)
	if path := os.Getenv("GITHUB_STEP_SUMMARY"); path != "" {
		f, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if openErr != nil {
			return errors.Join(err, openErr)
		}
		_, writeErr := f.WriteString(summary)
		return errors.Join(err, writeErr, f.Close())
	}
	return err
}
