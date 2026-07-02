package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type variantFlag []string

func (v *variantFlag) String() string {
	return strings.Join(*v, ";")
}

func (v *variantFlag) Set(s string) error {
	*v = append(*v, s)
	return nil
}

type variant struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Tool string `json:"tool"`
	Root string `json:"root,omitempty"`
	LTO  bool   `json:"lto,omitempty"`
}

type scenario struct {
	Name         string       `json:"name"`
	Kind         string       `json:"kind"`
	Dir          string       `json:"-"`
	PackageCount int          `json:"package_count,omitempty"`
	MethodCount  int          `json:"method_count,omitempty"`
	TargetCount  int          `json:"target_count,omitempty"`
	Scale        scenarioSize `json:"scale,omitempty"`
}

type scenarioSize struct {
	Packages int `json:"packages"`
	Methods  int `json:"methods"`
}

type buildResult struct {
	Variant  string `json:"variant"`
	Scenario string `json:"scenario"`
	Binary   string `json:"binary"`
	Size     int64  `json:"size_bytes"`
	BuildMS  int64  `json:"build_ms"`
	Error    string `json:"error,omitempty"`
}

type runResult struct {
	Variant  string             `json:"variant"`
	Scenario string             `json:"scenario"`
	Metrics  map[string][]int64 `json:"metrics_ns"`
	Error    string             `json:"error,omitempty"`
	Output   string             `json:"output,omitempty"`
	Env      map[string]string  `json:"env,omitempty"`
}

type resultFile struct {
	GeneratedAt  time.Time     `json:"generated_at"`
	PackageCount int           `json:"package_count"`
	MethodCount  int           `json:"method_count"`
	Variants     []variant     `json:"variants"`
	Scenarios    []string      `json:"scenarios"`
	ScenarioMeta []scenario    `json:"scenario_meta,omitempty"`
	Builds       []buildResult `json:"builds"`
	Runs         []runResult   `json:"runs"`
}

func main() {
	var variants variantFlag
	outDir := flag.String("out", filepath.Join("benchmark", "runtime_funcinfo", "out"), "output directory")
	runs := flag.Int("runs", 11, "process runs per executable")
	iters := flag.Int("iters", 200000, "inner benchmark iterations")
	llgoOpt := flag.String("llgo-opt", "2", "LLGo optimization level passed as -O<value>; empty disables the flag")
	scenarioList := flag.String("scenarios", "hot,deep,multipkg,cold,stdlib", "comma-separated scenarios")
	includeLTO := flag.Bool("include-lto", false, "also build full-LTO variants for LLGo compilers")
	pkgCount := flag.Int("packages", 12, "generated package count for multipkg")
	methodCount := flag.Int("methods", 12, "generated functions and methods per generated package")
	scaleList := flag.String("scales", "", "optional comma-separated package x method scales for multipkg/cold, for example 6x6,12x12,24x24")
	flag.Var(&variants, "variant", "variant definition: name=go or name=llgo,/path/to/llgo,/path/to/root")
	flag.Parse()

	if len(variants) == 0 {
		variants = append(variants, "go=go")
	}
	parsed, err := parseVariants(variants, *includeLTO)
	if err != nil {
		fatal(err)
	}
	if *runs <= 0 {
		fatal(errors.New("-runs must be positive"))
	}
	if *iters <= 0 {
		fatal(errors.New("-iters must be positive"))
	}
	if *pkgCount <= 0 || *methodCount <= 0 {
		fatal(errors.New("-packages and -methods must be positive"))
	}
	scales, err := parseScales(*scaleList)
	if err != nil {
		fatal(err)
	}

	absOut, err := filepath.Abs(*outDir)
	if err != nil {
		fatal(err)
	}
	if err := os.RemoveAll(absOut); err != nil {
		fatal(err)
	}
	for _, dir := range []string{"work", "bin"} {
		if err := os.MkdirAll(filepath.Join(absOut, dir), 0755); err != nil {
			fatal(err)
		}
	}

	scenarios, err := generateScenarios(filepath.Join(absOut, "work"), splitList(*scenarioList), *pkgCount, *methodCount, scales)
	if err != nil {
		fatal(err)
	}

	var builds []buildResult
	var runsOut []runResult
	for _, sc := range scenarios {
		for _, v := range parsed {
			br := buildScenario(absOut, sc, v, *llgoOpt)
			builds = append(builds, br)
			if br.Error != "" {
				fmt.Fprintf(os.Stderr, "build failed: %s/%s: %s\n", v.Name, sc.Name, br.Error)
				continue
			}
			rr := runScenario(sc, v, br.Binary, *runs, *iters)
			runsOut = append(runsOut, rr)
			if rr.Error != "" {
				fmt.Fprintf(os.Stderr, "run failed: %s/%s: %s\n", v.Name, sc.Name, rr.Error)
			}
		}
	}

	result := resultFile{
		GeneratedAt:  time.Now(),
		PackageCount: *pkgCount,
		MethodCount:  *methodCount,
		Variants:     parsed,
		Scenarios:    scenarioNames(scenarios),
		ScenarioMeta: scenarios,
		Builds:       builds,
		Runs:         runsOut,
	}
	if err := writeJSON(filepath.Join(absOut, "results.json"), result); err != nil {
		fatal(err)
	}
	summary := renderSummary(result)
	if err := os.WriteFile(filepath.Join(absOut, "summary.md"), []byte(summary), 0644); err != nil {
		fatal(err)
	}
	fmt.Print(summary)
}

func parseVariants(values []string, includeLTO bool) ([]variant, error) {
	var out []variant
	seen := map[string]bool{}
	for _, raw := range values {
		name, spec, ok := strings.Cut(raw, "=")
		if !ok || name == "" || spec == "" {
			return nil, fmt.Errorf("bad -variant %q", raw)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate variant %q", name)
		}
		seen[name] = true
		var v variant
		v.Name = name
		switch {
		case spec == "go":
			v.Kind = "go"
			v.Tool = "go"
		case strings.HasPrefix(spec, "go,"):
			parts := strings.Split(spec, ",")
			if len(parts) != 2 || parts[1] == "" {
				return nil, fmt.Errorf("bad go variant %q", raw)
			}
			v.Kind = "go"
			v.Tool = parts[1]
		case strings.HasPrefix(spec, "llgo,"):
			parts := strings.Split(spec, ",")
			if len(parts) != 3 || parts[1] == "" || parts[2] == "" {
				return nil, fmt.Errorf("bad llgo variant %q", raw)
			}
			v.Kind = "llgo"
			v.Tool = parts[1]
			v.Root = parts[2]
		default:
			return nil, fmt.Errorf("unknown variant kind in %q", raw)
		}
		out = append(out, v)
		if includeLTO && v.Kind == "llgo" {
			lto := v
			lto.Name = v.Name + "+lto"
			lto.LTO = true
			out = append(out, lto)
		}
	}
	return out, nil
}

func generateScenarios(workDir string, names []string, pkgCount, methodCount int, scales []scenarioSize) ([]scenario, error) {
	var out []scenario
	for _, name := range names {
		sizes := []scenarioSize{{Packages: pkgCount, Methods: methodCount}}
		if len(scales) != 0 && (name == "multipkg" || name == "cold") {
			sizes = scales
		}
		for _, size := range sizes {
			scenarioName := name
			if len(sizes) > 1 {
				scenarioName = fmt.Sprintf("%s_%dx%d", name, size.Packages, size.Methods)
			}
			dir := filepath.Join(workDir, scenarioName)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			var err error
			switch name {
			case "hot":
				err = generateHot(dir)
			case "deep":
				err = generateDeep(dir)
			case "multipkg":
				err = generateMultipkg(dir, size.Packages, size.Methods)
			case "cold":
				err = generateCold(dir, size.Packages, size.Methods)
			case "stdlib":
				err = generateStdlib(dir)
			default:
				return nil, fmt.Errorf("unknown scenario %q", name)
			}
			if err != nil {
				return nil, err
			}
			sc := scenario{Name: scenarioName, Kind: name, Dir: dir}
			if name == "multipkg" || name == "cold" {
				sc.PackageCount = size.Packages
				sc.MethodCount = size.Methods
				sc.TargetCount = size.Packages * size.Methods
				sc.Scale = size
			}
			out = append(out, sc)
		}
	}
	return out, nil
}

func writeModule(dir, module string) error {
	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.24\n"), 0644)
}

func generateHot(dir string) error {
	if err := writeModule(dir, "example.com/llgo-bench/hot"); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(hotSource), 0644)
}

func generateDeep(dir string) error {
	if err := writeModule(dir, "example.com/llgo-bench/deep"); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(deepPrefix)
	for i := 0; i < 32; i++ {
		fmt.Fprintf(&b, "//go:noinline\nfunc frame%d() { frame%d() }\n\n", i, i+1)
	}
	b.WriteString(`//go:noinline
func frame32() {
	pc, file, line, ok := runtime.Caller(16)
	if !ok || pc == 0 || file == "" || line == 0 {
		panic("bad deep caller")
	}
	sinkPC = pc
	sinkString = file
	sinkInt += line
}

`)
	b.WriteString(deepSuffix)
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(b.String()), 0644)
}

func generateMultipkg(dir string, pkgCount, methodCount int) error {
	if err := writeModule(dir, "example.com/llgo-bench/multipkg"); err != nil {
		return err
	}
	for i := 0; i < pkgCount; i++ {
		pkgName := fmt.Sprintf("p%02d", i)
		pkgDir := filepath.Join(dir, pkgName)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "package %s\n\n", pkgName)
		b.WriteString("import (\n\t\"reflect\"\n\t\"runtime\"\n")
		if i+1 < pkgCount {
			fmt.Fprintf(&b, "\tnext \"example.com/llgo-bench/multipkg/p%02d\"\n", i+1)
		}
		b.WriteString(")\n\n")
		fmt.Fprintf(&b, "type T%02d struct { V int }\n", i)
		b.WriteString("type Worker interface { M00(int) int }\n\n")
		for j := 0; j < methodCount; j++ {
			fmt.Fprintf(&b, "//go:noinline\nfunc F%02d_%02d(x int) int { return x + %d }\n\n", i, j, i*100+j)
			fmt.Fprintf(&b, "//go:noinline\nfunc (t T%02d) M%02d(x int) int { return t.V + x + %d }\n\n", i, j, j)
		}
		b.WriteString("//go:noinline\nfunc Targets() []uintptr {\n\treturn []uintptr{\n")
		for j := 0; j < methodCount; j++ {
			fmt.Fprintf(&b, "\t\treflect.ValueOf(F%02d_%02d).Pointer(),\n", i, j)
		}
		b.WriteString("\t}\n}\n\n")
		b.WriteString("//go:noinline\nfunc Run(x int) int {\n")
		b.WriteString("\tpc, _, line, ok := runtime.Caller(0)\n\tif !ok || pc == 0 || line == 0 { panic(\"bad caller\") }\n")
		fmt.Fprintf(&b, "\tvar w Worker = T%02d{V: x}\n", i)
		b.WriteString("\ttotal := w.M00(x)\n")
		for j := 0; j < methodCount; j++ {
			fmt.Fprintf(&b, "\ttotal += F%02d_%02d(x)\n", i, j)
			fmt.Fprintf(&b, "\ttotal += (T%02d{V: total}).M%02d(x)\n", i, j)
		}
		if i+1 < pkgCount {
			b.WriteString("\ttotal += next.Run(x+1)\n")
		}
		b.WriteString("\treturn total + line\n}\n")
		if err := os.WriteFile(filepath.Join(pkgDir, pkgName+".go"), []byte(b.String()), 0644); err != nil {
			return err
		}
	}

	var main strings.Builder
	main.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"runtime\"\n\t\"time\"\n")
	for i := 0; i < pkgCount; i++ {
		fmt.Fprintf(&main, "\tp%02d \"example.com/llgo-bench/multipkg/p%02d\"\n", i, i)
	}
	main.WriteString(")\n\nvar sinkInt int\nvar sinkString string\n\n")
	main.WriteString(commonBenchHelpers)
	main.WriteString("func main() {\n\titers := benchIters(10000)\n\tvar targets []uintptr\n")
	for i := 0; i < pkgCount; i++ {
		fmt.Fprintf(&main, "\ttargets = append(targets, p%02d.Targets()...)\n", i)
	}
	main.WriteString(`
	if funcInfoReady(targets) {
		measure("multipkg.FuncForPCMany", iters, func() {
			total := 0
			for _, pc := range targets {
				fn := runtime.FuncForPC(pc)
				if fn == nil {
					panic("missing func")
				}
				total += len(fn.Name())
			}
			sinkInt += total
		})
		measure("multipkg.FileLineMany", iters, func() {
			total := 0
			for _, pc := range targets {
				fn := runtime.FuncForPC(pc)
				if fn == nil {
					panic("missing func")
				}
				file, line := fn.FileLine(pc)
				if file == "" || line == 0 {
					panic("missing fileline")
				}
				total += line + len(file)
			}
			sinkInt += total
		})
	}
	measure("multipkg.DeepRun", iters, func() {
		sinkInt += p00.Run(1)
	})
	fmt.Println("sink=", sinkInt, sinkString)
}

func funcInfoReady(targets []uintptr) bool {
	for _, pc := range targets {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			return false
		}
		if file, line := fn.FileLine(pc); file == "" || line == 0 {
			return false
		}
	}
	return len(targets) != 0
}
`)
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(main.String()), 0644)
}

func generateCold(dir string, pkgCount, methodCount int) error {
	if err := writeModule(dir, "example.com/llgo-bench/cold"); err != nil {
		return err
	}
	for i := 0; i < pkgCount; i++ {
		pkgName := fmt.Sprintf("p%02d", i)
		pkgDir := filepath.Join(dir, pkgName)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "package %s\n\n", pkgName)
		b.WriteString("import \"reflect\"\n\n")
		for j := 0; j < methodCount; j++ {
			fmt.Fprintf(&b, "//go:noinline\nfunc F%02d_%02d(x int) int { return x + %d }\n\n", i, j, i*100+j)
		}
		b.WriteString("//go:noinline\nfunc Targets() []uintptr {\n\treturn []uintptr{\n")
		for j := 0; j < methodCount; j++ {
			fmt.Fprintf(&b, "\t\treflect.ValueOf(F%02d_%02d).Pointer(),\n", i, j)
		}
		b.WriteString("\t}\n}\n")
		if err := os.WriteFile(filepath.Join(pkgDir, pkgName+".go"), []byte(b.String()), 0644); err != nil {
			return err
		}
	}

	var main strings.Builder
	main.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"runtime\"\n\t\"time\"\n")
	for i := 0; i < pkgCount; i++ {
		fmt.Fprintf(&main, "\tp%02d \"example.com/llgo-bench/cold/p%02d\"\n", i, i)
	}
	main.WriteString(")\n\nvar sinkInt int\nvar sinkString string\n\n")
	main.WriteString(commonBenchHelpers)
	main.WriteString("func main() {\n\titers := benchIters(10000)\n\tvar targets []uintptr\n")
	for i := 0; i < pkgCount; i++ {
		fmt.Fprintf(&main, "\ttargets = append(targets, p%02d.Targets()...)\n", i)
	}
	main.WriteString(`
	if len(targets) == 0 {
		panic("missing targets")
	}
	first := targets[len(targets)/2]
	start := time.Now()
	fn := runtime.FuncForPC(first)
	if fn == nil || fn.Name() == "" {
		panic("missing first func")
	}
	fmt.Printf("cold.FirstFuncForPC=%d\n", time.Since(start).Nanoseconds())
	sinkString = fn.Name()

	start = time.Now()
	file, line := fn.FileLine(first)
	if file == "" || line == 0 {
		panic("missing first fileline")
	}
	fmt.Printf("cold.FirstFileLine=%d\n", time.Since(start).Nanoseconds())
	sinkString = file
	sinkInt += line

	start = time.Now()
	pc, file, line, ok := runtime.Caller(0)
	if !ok || pc == 0 || file == "" || line == 0 {
		panic("bad first caller")
	}
	fmt.Printf("cold.FirstCaller0=%d\n", time.Since(start).Nanoseconds())
	sinkString = file
	sinkInt += line

	start = time.Now()
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	// Walk to the first fully symbolized frame: synthetic runtime frames
	// (e.g. LLGo's runtime.Callers placeholder) carry no file/line.
	for {
		frame, more := frames.Next()
		if frame.Function != "" && frame.File != "" && frame.Line != 0 {
			fmt.Printf("cold.FirstCallersFrames=%d\n", time.Since(start).Nanoseconds())
			sinkString = frame.Function
			sinkInt += frame.Line
			break
		}
		if !more {
			break
		}
	}

	measure("cold.WarmFuncForPCMany", iters, func() {
		total := 0
		for _, pc := range targets {
			fn := runtime.FuncForPC(pc)
			if fn == nil {
				panic("missing func")
			}
			total += len(fn.Name())
		}
		sinkInt += total
	})
	measure("cold.WarmFileLineMany", iters, func() {
		total := 0
		for _, pc := range targets {
			fn := runtime.FuncForPC(pc)
			if fn == nil {
				panic("missing func")
			}
			file, line := fn.FileLine(pc)
			if file == "" || line == 0 {
				panic("missing fileline")
			}
			total += len(file) + line
		}
		sinkInt += total
	})
	fmt.Println("sink=", sinkInt, sinkString)
}
`)
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(main.String()), 0644)
}

func generateStdlib(dir string) error {
	if err := writeModule(dir, "example.com/llgo-bench/stdlib"); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(stdlibSource), 0644)
}

func buildScenario(outDir string, sc scenario, v variant, llgoOpt string) buildResult {
	bin := filepath.Join(outDir, "bin", safeName(v.Name)+"_"+sc.Name)
	if v.LTO {
		bin += "_lto"
	}
	if exeSuffix := executableSuffix(); exeSuffix != "" {
		bin += exeSuffix
	}
	start := time.Now()
	var cmd *exec.Cmd
	switch v.Kind {
	case "go":
		cmd = exec.Command(v.Tool, "build", "-trimpath", "-o", bin, ".")
	case "llgo":
		args := []string{"build", "-trimpath", "-a", "-o", bin}
		if llgoOpt != "" {
			args = append(args, "-O"+llgoOpt)
		}
		if v.LTO {
			args = append(args, "-lto=full")
		}
		args = append(args, ".")
		cmd = exec.Command(v.Tool, args...)
	default:
		return buildResult{Variant: v.Name, Scenario: sc.Name, Binary: bin, Error: "unknown variant kind"}
	}
	cmd.Dir = sc.Dir
	cmd.Env = os.Environ()
	if v.Kind == "llgo" {
		cmd.Env = append(cmd.Env, "LLGO_ROOT="+v.Root, "LLGO_FUNCINFO=1")
	}
	out, err := cmd.CombinedOutput()
	br := buildResult{Variant: v.Name, Scenario: sc.Name, Binary: bin, BuildMS: time.Since(start).Milliseconds()}
	if err != nil {
		br.Error = strings.TrimSpace(string(out))
		if br.Error == "" {
			br.Error = err.Error()
		}
		return br
	}
	info, err := os.Stat(bin)
	if err != nil {
		br.Error = err.Error()
		return br
	}
	br.Size = info.Size()
	return br
}

func runScenario(sc scenario, v variant, bin string, runs, iters int) runResult {
	scenarioIters := iterationsForScenario(sc.Kind, iters)
	rr := runResult{
		Variant:  v.Name,
		Scenario: sc.Name,
		Metrics:  map[string][]int64{},
		Env: map[string]string{
			"BENCH_ITERS": strconv.Itoa(scenarioIters),
		},
	}
	for i := 0; i < runs; i++ {
		cmd := exec.Command(bin)
		cmd.Dir = sc.Dir
		cmd.Env = append(os.Environ(), "BENCH_ITERS="+strconv.Itoa(scenarioIters))
		out, err := cmd.CombinedOutput()
		if err != nil {
			rr.Error = err.Error()
			rr.Output = string(out)
			return rr
		}
		metrics, err := parseMetrics(out)
		if err != nil {
			rr.Error = err.Error()
			rr.Output = string(out)
			return rr
		}
		for k, v := range metrics {
			rr.Metrics[k] = append(rr.Metrics[k], v)
		}
	}
	return rr
}

func iterationsForScenario(name string, base int) int {
	div := 1
	switch name {
	case "deep":
		div = 4
	case "multipkg", "cold", "stdlib":
		div = 20
	}
	n := base / div
	if n < 1 {
		return 1
	}
	return n
}

func parseMetrics(out []byte) (map[string]int64, error) {
	metrics := map[string]int64{}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		name, value, _ := strings.Cut(line, "=")
		if !strings.Contains(name, ".") {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse metric %q: %w", line, err)
		}
		metrics[strings.TrimSpace(name)] = n
	}
	return metrics, nil
}

func renderSummary(result resultFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Runtime Funcinfo Benchmark\n\nGenerated: `%s`\n\n", result.GeneratedAt.Format(time.RFC3339))
	b.WriteString("Cells are `best/trimmed avg`. Runtime metrics use `ns/op`; sizes use MiB.\n\n")
	for _, sc := range result.ScenarioMeta {
		if sc.TargetCount == 0 {
			continue
		}
		switch sc.Kind {
		case "multipkg":
			fmt.Fprintf(&b, "`%s` uses `multipkg.FuncForPCMany` and `multipkg.FileLineMany` batch metrics over %d target functions (%d packages x %d functions).\n\n",
				sc.Name, sc.TargetCount, sc.PackageCount, sc.MethodCount)
		case "cold":
			fmt.Fprintf(&b, "`%s` uses `cold.WarmFuncForPCMany` and `cold.WarmFileLineMany` batch metrics over %d target functions (%d packages x %d functions). `cold.First*` metrics are one per process and include lazy runtime initialization that has not already happened in that process.\n\n",
				sc.Name, sc.TargetCount, sc.PackageCount, sc.MethodCount)
		}
	}
	for _, sc := range result.Scenarios {
		metrics := metricsForScenario(result.Runs, sc)
		if len(metrics) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s Performance\n\n", sc)
		b.WriteString("| metric |")
		for _, v := range result.Variants {
			b.WriteString(" " + v.Name + " |")
		}
		b.WriteString("\n|---|")
		for range result.Variants {
			b.WriteString("---:|")
		}
		b.WriteString("\n")
		for _, metric := range metrics {
			b.WriteString("| " + metric + " |")
			for _, v := range result.Variants {
				rr, found := findRun(result.Runs, v.Name, sc)
				cell := "FAIL"
				if found && rr.Error == "" {
					cell = "n/a"
				}
				if vals := rr.Metrics[metric]; len(vals) != 0 {
					cell = formatPerf(vals)
				}
				b.WriteString(" " + cell + " |")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Binary Size\n\n| scenario |")
	for _, v := range result.Variants {
		b.WriteString(" " + v.Name + " |")
	}
	b.WriteString("\n|---|")
	for range result.Variants {
		b.WriteString("---:|")
	}
	b.WriteString("\n")
	for _, sc := range result.Scenarios {
		b.WriteString("| " + sc + " |")
		for _, v := range result.Variants {
			cell := "FAIL"
			if br := findBuild(result.Builds, v.Name, sc); br.Error == "" && br.Size > 0 {
				cell = formatMiB(br.Size)
			}
			b.WriteString(" " + cell + " |")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Build Time\n\n| scenario |")
	for _, v := range result.Variants {
		b.WriteString(" " + v.Name + " |")
	}
	b.WriteString("\n|---|")
	for range result.Variants {
		b.WriteString("---:|")
	}
	b.WriteString("\n")
	for _, sc := range result.Scenarios {
		b.WriteString("| " + sc + " |")
		for _, v := range result.Variants {
			cell := "FAIL"
			if br := findBuild(result.Builds, v.Name, sc); br.Error == "" {
				cell = formatDurationMS(br.BuildMS)
			}
			b.WriteString(" " + cell + " |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func metricsForScenario(runs []runResult, scenario string) []string {
	set := map[string]bool{}
	for _, rr := range runs {
		if rr.Scenario != scenario || rr.Error != "" {
			continue
		}
		for k := range rr.Metrics {
			set[k] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func findRun(runs []runResult, variant, scenario string) (runResult, bool) {
	for _, rr := range runs {
		if rr.Variant == variant && rr.Scenario == scenario {
			return rr, true
		}
	}
	return runResult{Metrics: map[string][]int64{}}, false
}

func findBuild(builds []buildResult, variant, scenario string) buildResult {
	for _, br := range builds {
		if br.Variant == variant && br.Scenario == scenario {
			return br
		}
	}
	return buildResult{Error: "missing"}
}

func formatPerf(values []int64) string {
	if len(values) == 0 {
		return "n/a"
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	best := sorted[0]
	avgVals := sorted
	if len(sorted) >= 3 {
		avgVals = sorted[1 : len(sorted)-1]
	}
	var sum int64
	for _, v := range avgVals {
		sum += v
	}
	avg := float64(sum) / float64(len(avgVals))
	return formatNS(float64(best)) + "/" + formatNS(avg)
}

func formatNS(ns float64) string {
	switch {
	case ns >= 1e6:
		return trimFloat(ns/1e6) + "ms"
	case ns >= 1e3:
		return trimFloat(ns/1e3) + "us"
	default:
		return trimFloat(ns) + "ns"
	}
}

func formatMiB(bytes int64) string {
	return trimFloat(float64(bytes)/(1024*1024)) + " MiB"
}

func formatDurationMS(ms int64) string {
	if ms >= 1000 {
		return trimFloat(float64(ms)/1000) + "s"
	}
	return strconv.FormatInt(ms, 10) + "ms"
}

func trimFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.05 {
		return strconv.FormatInt(int64(math.Round(v)), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func writeJSON(path string, data any) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0644)
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseScales(s string) ([]scenarioSize, error) {
	var out []scenarioSize
	for _, part := range splitList(s) {
		left, right, ok := strings.Cut(part, "x")
		if !ok {
			left, right, ok = strings.Cut(part, "X")
		}
		if !ok {
			return nil, fmt.Errorf("bad scale %q: want packages x methods, for example 12x12", part)
		}
		packages, err := strconv.Atoi(strings.TrimSpace(left))
		if err != nil || packages <= 0 {
			return nil, fmt.Errorf("bad package count in scale %q", part)
		}
		methods, err := strconv.Atoi(strings.TrimSpace(right))
		if err != nil || methods <= 0 {
			return nil, fmt.Errorf("bad method count in scale %q", part)
		}
		out = append(out, scenarioSize{Packages: packages, Methods: methods})
	}
	return out, nil
}

func scenarioNames(scenarios []scenario) []string {
	out := make([]string, len(scenarios))
	for i, sc := range scenarios {
		out[i] = sc.Name
	}
	return out
}

func safeName(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "+", "_")
	return replacer.Replace(s)
}

func executableSuffix() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

const commonBenchHelpers = `
func benchIters(def int) int {
	if s := getenv("BENCH_ITERS"); s != "" {
		n, err := atoi(s)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func measure(name string, n int, fn func()) {
	fn()
	start := time.Now()
	for i := 0; i < n; i++ {
		fn()
	}
	elapsed := time.Since(start).Nanoseconds()
	if n <= 0 {
		panic("bad iterations")
	}
	fmt.Printf("%s=%d\n", name, elapsed/int64(n))
}

func getenv(k string) string {
	for _, kv := range os.Environ() {
		if len(kv) > len(k) && kv[:len(k)] == k && kv[len(k)] == '=' {
			return kv[len(k)+1:]
		}
	}
	return ""
}

func atoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("bad int")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}
`

const hotSource = `package main

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"time"
)

var sinkInt int
var sinkPC uintptr
var sinkString string

` + commonBenchHelpers + `

//go:noinline
func entryTarget(x int) int {
	return x + 7
}

//go:noinline
func caller0() {
	pc, file, line, ok := runtime.Caller(0)
	if !ok || pc == 0 || file == "" || line == 0 {
		panic("bad caller0")
	}
	sinkPC = pc
	sinkString = file
	sinkInt += line
}

//go:noinline
func caller1() {
	caller1Helper()
}

//go:noinline
func caller1Helper() {
	pc, file, line, ok := runtime.Caller(1)
	if !ok || pc == 0 || file == "" || line == 0 {
		panic("bad caller1")
	}
	sinkPC = pc
	sinkString = file
	sinkInt += line
}

//go:noinline
func returnPC() uintptr {
	pc, _, _, ok := runtime.Caller(0)
	if !ok || pc == 0 {
		panic("bad return pc")
	}
	return pc
}

//go:noinline
func callersOnly() {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	if n == 0 || pcs[0] == 0 {
		panic("bad callers")
	}
	sinkPC = pcs[0]
	sinkInt += n
}

//go:noinline
func callersFramesFirst() {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function != "" && frame.File != "" && frame.Line != 0 {
			sinkString = frame.Function
			sinkInt += frame.Line
			return
		}
		if !more {
			break
		}
	}
	panic("bad frame")
}

func callersFramesReady() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	callersFramesFirst()
	return true
}

func main() {
	iters := benchIters(200000)
	entryPC := reflect.ValueOf(entryTarget).Pointer()
	returnedPC := returnPC()
	measure("hot.Caller0", iters, caller0)
	measure("hot.Caller1", iters, caller1)
	measure("hot.CallersOnly", iters, callersOnly)
	if callersFramesReady() {
		measure("hot.CallersFramesFirst", iters, callersFramesFirst)
	}
	if entryFn := runtime.FuncForPC(entryPC); entryFn != nil && entryFn.Name() != "" {
		measure("hot.FuncForPCEntry", iters, func() {
			fn := runtime.FuncForPC(entryPC)
			if fn == nil {
				panic("missing entry func")
			}
			sinkString = fn.Name()
		})
		if file, line := entryFn.FileLine(entryPC); file != "" && line != 0 {
			measure("hot.FuncFileLineEntry", iters, func() {
				file, line := entryFn.FileLine(entryPC)
				if file == "" || line == 0 {
					panic("missing entry fileline")
				}
				sinkString = file
				sinkInt += line
			})
		}
	}
	if returnFn := runtime.FuncForPC(returnedPC); returnFn != nil && returnFn.Name() != "" {
		measure("hot.FuncForPCReturnPC", iters, func() {
			fn := runtime.FuncForPC(returnedPC)
			if fn == nil {
				panic("missing return func")
			}
			sinkString = fn.Name()
		})
		if file, line := returnFn.FileLine(returnedPC); file != "" && line != 0 {
			measure("hot.FuncFileLineReturnPC", iters, func() {
				file, line := returnFn.FileLine(returnedPC)
				if file == "" || line == 0 {
					panic("missing return fileline")
				}
				sinkString = file
				sinkInt += line
			})
		}
	}
	fmt.Println("sink=", sinkInt, sinkPC, sinkString)
}
`

const deepPrefix = `package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var sinkInt int
var sinkPC uintptr
var sinkString string

` + commonBenchHelpers + `

type callerIface interface {
	call()
}

type callerImpl struct{}

//go:noinline
func (callerImpl) call() {
	frame0()
}

//go:noinline
func closureLayer(next func()) func() {
	return func() {
		next()
	}
}

//go:noinline
func callInterface(c callerIface) {
	c.call()
}

//go:noinline
func callClosure() {
	closureLayer(closureLayer(frame0))()
}

`

const deepSuffix = `//go:noinline
func framesAll() {
	frame0()
	var pcs [64]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	total := 0
	for {
		frame, more := frames.Next()
		if frame.Function != "" {
			total += len(frame.Function) + frame.Line
		}
		if !more {
			break
		}
	}
	if total == 0 {
		panic("bad frames")
	}
	sinkInt += total
}

func deepReady(fn func()) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	fn()
	return true
}

func main() {
	iters := benchIters(50000)
	if deepReady(frame0) {
		measure("deep.Direct32", iters, frame0)
	}
	if deepReady(func() { callInterface(callerImpl{}) }) {
		measure("deep.Interface32", iters, func() { callInterface(callerImpl{}) })
	}
	if deepReady(callClosure) {
		measure("deep.Closure32", iters, callClosure)
	}
	if deepReady(framesAll) {
		measure("deep.CallersFramesAll", iters, framesAll)
	}
	fmt.Println("sink=", sinkInt, sinkPC, sinkString)
}
`

const stdlibSource = `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"net/netip"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"
)

var sinkInt int
var sinkString string

` + commonBenchHelpers + `

type payload struct {
	Name  string
	Items []int
	Addr  string
}

//go:noinline
func stdTarget(x int) int {
	return x*3 + 1
}

//go:noinline
func stdWork() {
	p := payload{Name: "llgo", Items: []int{1, 2, 3, 5, 8}, Addr: "127.0.0.1:8080"}
	raw, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	var out payload
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	tmpl := template.Must(template.New("x").Funcs(template.FuncMap{"join": strings.Join}).Parse("{{.Name}}:{{join .Words \",\"}}"))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{"Name": out.Name, "Words": []string{"a", "b", "c"}}); err != nil {
		panic(err)
	}
	re := regexp.MustCompile("[a-z]+")
	matches := re.FindAllString(buf.String(), -1)
	expr, err := parser.ParseExpr("1 + 2*3")
	if err != nil || expr == nil {
		panic("bad parser")
	}
	fs := token.NewFileSet()
	file := fs.AddFile("bench.go", -1, 100)
	file.AddLine(10)
	addr := netip.MustParseAddrPort(out.Addr)
	sinkInt += len(matches) + int(addr.Port()) + int(file.Line(token.Pos(11)))
	sinkString = buf.String()
}

//go:noinline
func stdCaller() {
	pc, file, line, ok := runtime.Caller(0)
	if !ok || pc == 0 || file == "" || line == 0 {
		panic("bad caller")
	}
	sinkInt += line
	sinkString = file
}

//go:noinline
func stdFrames() {
	var pcs [16]uintptr
	n := runtime.Callers(0, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function != "" && frame.File != "" && frame.Line != 0 {
			sinkInt += frame.Line
			sinkString = frame.Function
			return
		}
		if !more {
			break
		}
	}
	panic("bad frame")
}

func stdFramesReady() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	stdFrames()
	return true
}

func main() {
	iters := benchIters(50000)
	entryPC := reflect.ValueOf(stdTarget).Pointer()
	measure("stdlib.Work", iters/10, stdWork)
	measure("stdlib.Caller0", iters, stdCaller)
	if stdFramesReady() {
		measure("stdlib.CallersFramesFirst", iters, stdFrames)
	}
	if fn := runtime.FuncForPC(entryPC); fn != nil && fn.Name() != "" {
		measure("stdlib.FuncForPCEntry", iters, func() {
			fn := runtime.FuncForPC(entryPC)
			if fn == nil {
				panic("missing func")
			}
			sinkString = fn.Name()
		})
		if file, line := fn.FileLine(entryPC); file != "" && line != 0 {
			measure("stdlib.FuncFileLineEntry", iters, func() {
				file, line := fn.FileLine(entryPC)
				if file == "" || line == 0 {
					panic("missing fileline")
				}
				sinkInt += line
				sinkString = file
			})
		}
	}
	fmt.Println("sink=", sinkInt, sinkString)
}
`
