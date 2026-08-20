/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"bytes"
	"context"
	"debug/elf"
	"debug/macho"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type metric struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
	Range string  `json:"range,omitempty"`
	Extra string  `json:"extra,omitempty"`
}

type workload struct {
	name             string
	source           string
	output           string
	args             []string
	harnessSource    bool
	internalDuration bool
}

var workloads = []workload{
	{name: "cprintf", source: "benchmark/binary_size/cprintf/main.go", output: "Hello, world\n"},
	{name: "println", source: "benchmark/binary_size/println/main.go", output: "Hello, world\n"},
	{name: "fmtprintf", source: "benchmark/binary_size/fmtprintf/main.go", output: "Hello, world\n"},
	{
		name:             "memprofile-no-consumer",
		source:           "benchmark/memprofile/noconsumer",
		harnessSource:    true,
		internalDuration: true,
	},
	{
		name:             "memprofile-rate0",
		source:           "benchmark/memprofile/enabled",
		args:             []string{"rate0"},
		harnessSource:    true,
		internalDuration: true,
	},
	{
		name:             "memprofile-default",
		source:           "benchmark/memprofile/enabled",
		harnessSource:    true,
		internalDuration: true,
	},
}

var expectedGoBenchmarks = []string{
	"BenchmarkChannelBuffered",
	"BenchmarkChannelHandoff",
	"BenchmarkDefer",
	"BenchmarkDirectCall",
	"BenchmarkGlobalRead",
	"BenchmarkGlobalWrite",
	"BenchmarkGoroutine",
	"BenchmarkInterfaceCall",
	"BenchmarkLookupPCRandom",
	"BenchmarkMergeCompilerFlags",
	"BenchmarkMergeLinkerFlags",
	"BenchmarkRuntimeGetG",
}

const goBenchmarkSamples = 7

type footprint struct {
	file uint64
	text uint64
	data uint64
	bss  uint64
}

var inspectExecutable = executableFootprint

func main() {
	if err := runCLI(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("llgo-baseline", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "collect", "collect, collect-paired, validate, or export")
	root := flags.String("root", ".", "LLGo repository root")
	harnessRoot := flags.String("harness-root", ".", "benchmark harness repository root")
	llgo := flags.String("llgo", "llgo", "LLGo command")
	out := flags.String("out", filepath.Join("benchmark", "baseline", "out"), "result directory")
	baseRoot := flags.String("base-root", "", "comparison LLGo repository root")
	baseLLGo := flags.String("base-llgo", "", "comparison LLGo command")
	baseOut := flags.String("base-out", "", "comparison result directory")
	buildRuns := flags.Int("build-runs", 6, "build repetitions per workload")
	runRuns := flags.Int("run-runs", 18, "process repetitions per workload")
	benchmarkOutput := flags.String(
		"benchmark-output",
		"",
		"standard Go benchmark output for export mode",
	)
	if err := flags.Parse(args); err != nil {
		return err
	}

	switch *mode {
	case "collect":
		return collectWithHarness(ctx, *root, *harnessRoot, *llgo, *out, *buildRuns, *runRuns)
	case "collect-paired":
		if *baseRoot == "" || *baseLLGo == "" || *baseOut == "" {
			return errors.New("collect-paired mode requires base-root, base-llgo, and base-out")
		}
		return collectPaired(
			ctx,
			collectionSpec{name: "base", root: *baseRoot, harnessRoot: *harnessRoot, llgo: *baseLLGo, out: *baseOut},
			collectionSpec{name: "current", root: *root, harnessRoot: *harnessRoot, llgo: *llgo, out: *out},
			*buildRuns,
			*runRuns,
		)
	case "validate":
		return validateArtifact(*out)
	case "export":
		return exportBenchmarks(*out, *benchmarkOutput)
	default:
		return fmt.Errorf("unknown mode %q", *mode)
	}
}

func exportBenchmarks(dir, output string) error {
	if output == "" {
		return errors.New("export mode requires benchmark-output")
	}
	if err := validateArtifact(dir); err != nil {
		return err
	}
	sizes, err := readMetrics(filepath.Join(dir, "size.json"))
	if err != nil {
		return err
	}
	timings, err := readMetrics(filepath.Join(dir, "time.json"))
	if err != nil {
		return err
	}
	core, err := os.ReadFile(filepath.Join(dir, "go.txt"))
	if err != nil {
		return err
	}
	byName := make(map[string]float64, len(sizes)+len(timings))
	for _, value := range append(sizes, timings...) {
		byName[value.Name] = value.Value
	}

	var data strings.Builder
	fmt.Fprintf(&data, "goos: %s\ngoarch: %s\n", runtime.GOOS, runtime.GOARCH)
	data.WriteString("pkg: github.com/xgo-dev/llgo/benchmark/baseline\n")
	for _, unit := range []string{
		"file-bytes",
		"text-bytes",
		"data-bytes",
		"bss-bytes",
	} {
		fmt.Fprintf(&data, "Unit %s better=lower assume=exact\n", unit)
	}
	data.WriteString("Unit build-ns better=lower\n")
	data.WriteString("Unit run-ns better=lower\n")
	for _, item := range workloads {
		fmt.Fprintf(
			&data,
			"BenchmarkProgram/%s 1 %s file-bytes %s text-bytes %s data-bytes %s bss-bytes %s build-ns %s run-ns\n",
			item.name,
			formatMetric(byName["binary/"+item.name+"/file"]),
			formatMetric(byName["binary/"+item.name+"/text"]),
			formatMetric(byName["binary/"+item.name+"/data"]),
			formatMetric(byName["binary/"+item.name+"/bss"]),
			formatMetric(byName["compile/"+item.name]),
			formatMetric(byName["run/"+item.name]),
		)
	}
	if len(core) > 0 && core[len(core)-1] != '\n' {
		core = append(core, '\n')
	}
	data.Write(core)
	return os.WriteFile(output, []byte(data.String()), 0o644)
}

func readMetrics(path string) ([]metric, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var values []metric
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return values, nil
}

func formatMetric(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func collect(ctx context.Context, root, llgo, out string, buildRuns, runRuns int) error {
	return collectWithHarness(ctx, root, root, llgo, out, buildRuns, runRuns)
}

func collectWithHarness(
	ctx context.Context, root, harnessRoot, llgo, out string, buildRuns, runRuns int,
) error {
	return collectSpecs(ctx, []collectionSpec{{
		root: root, harnessRoot: harnessRoot, llgo: llgo, out: out,
	}}, buildRuns, runRuns)
}

type collectionSpec struct {
	name        string
	root        string
	harnessRoot string
	llgo        string
	out         string
}

type workloadMeasurement struct {
	binary         string
	buildDurations []time.Duration
	runDurations   []time.Duration
}

type collectionState struct {
	collectionSpec
	binDir       string
	env          []string
	measurements []workloadMeasurement
}

func collectPaired(
	ctx context.Context, base, current collectionSpec, buildRuns, runRuns int,
) error {
	return collectSpecs(ctx, []collectionSpec{base, current}, buildRuns, runRuns)
}

func collectSpecs(ctx context.Context, specs []collectionSpec, buildRuns, runRuns int) error {
	if buildRuns <= 0 || runRuns <= 0 {
		return errors.New("build and run repetitions must be positive")
	}
	if len(specs) == 0 {
		return errors.New("at least one collection is required")
	}
	states := make([]collectionState, len(specs))
	for index, spec := range specs {
		root, err := filepath.Abs(spec.root)
		if err != nil {
			return err
		}
		harnessRoot, err := filepath.Abs(spec.harnessRoot)
		if err != nil {
			return err
		}
		out, err := filepath.Abs(spec.out)
		if err != nil {
			return err
		}
		binDir := filepath.Join(out, "bin")
		if err := os.RemoveAll(out); err != nil {
			return err
		}
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			return err
		}
		states[index] = collectionState{
			collectionSpec: collectionSpec{
				name: spec.name, root: root, harnessRoot: harnessRoot, llgo: spec.llgo, out: out,
			},
			binDir:       binDir,
			env:          append(os.Environ(), "GOMAXPROCS=2", "LLGO_ROOT="+root, "LLGO_FULL_RPATH=true"),
			measurements: make([]workloadMeasurement, len(workloads)),
		}
	}

	// Warm every workload before measuring any of them so the first workload
	// does not pay unique toolchain and filesystem cache costs. Paired
	// collections alternate which revision runs first for adjacent workloads.
	for workloadIndex, item := range workloads {
		for stateOffset := range states {
			state := &states[collectionStateIndex(0, workloadIndex, stateOffset, len(states))]
			if err := buildWorkload(ctx, state, item); err != nil {
				return collectionError(state, "warm build", item.name, err)
			}
		}
	}

	// Rotate both workload and revision order across rounds. In paired mode each
	// base/current measurement is adjacent, so runner drift is not attributed to
	// one complete revision phase.
	for round := range buildRuns {
		for offset := range workloads {
			index := (round + offset) % len(workloads)
			item := workloads[index]
			for stateOffset := range states {
				state := &states[collectionStateIndex(round, index, stateOffset, len(states))]
				start := time.Now()
				if err := buildWorkload(ctx, state, item); err != nil {
					return collectionError(state, "build", item.name, err)
				}
				state.measurements[index].buildDurations = append(
					state.measurements[index].buildDurations, time.Since(start),
				)
			}
		}
	}

	// Inspect final binaries and warm every execution path before timing.
	for workloadIndex, item := range workloads {
		for stateOffset := range states {
			state := &states[collectionStateIndex(0, workloadIndex, stateOffset, len(states))]
			binary := filepath.Join(state.binDir, item.name)
			state.measurements[workloadIndex].binary = binary
			if _, err := inspectExecutable(binary); err != nil {
				return collectionError(state, "inspect", item.name, err)
			}
			if _, err := executeWorkload(ctx, state, item, binary); err != nil {
				return collectionError(state, "execute", item.name, err)
			}
		}
	}

	for round := range runRuns {
		for offset := range workloads {
			index := (round + offset) % len(workloads)
			item := workloads[index]
			for stateOffset := range states {
				state := &states[collectionStateIndex(round, index, stateOffset, len(states))]
				measurement := &state.measurements[index]
				duration, err := executeWorkload(ctx, state, item, measurement.binary)
				if err != nil {
					return collectionError(state, "execute", item.name, err)
				}
				measurement.runDurations = append(measurement.runDurations, duration)
			}
		}
	}

	for stateIndex := range states {
		if err := writeCollection(&states[stateIndex]); err != nil {
			return err
		}
	}
	return nil
}

func collectionStateIndex(round, workloadIndex, stateOffset, stateCount int) int {
	return (round + workloadIndex + stateOffset) % stateCount
}

func buildWorkload(ctx context.Context, state *collectionState, item workload) error {
	binary := filepath.Join(state.binDir, item.name)
	sourceRoot := state.root
	if item.harnessSource {
		sourceRoot = state.harnessRoot
	}
	return run(ctx, state.env, io.Discard, state.llgo, "build", "-o", binary, filepath.Join(sourceRoot, item.source))
}

func executeWorkload(
	ctx context.Context, state *collectionState, item workload, binary string,
) (time.Duration, error) {
	var output bytes.Buffer
	start := time.Now()
	if err := run(ctx, state.env, &output, binary, item.args...); err != nil {
		return 0, err
	}
	duration := time.Since(start)
	if item.internalDuration {
		return parseInternalDuration(output.String())
	}
	if got := strings.ReplaceAll(output.String(), "\r\n", "\n"); got != item.output {
		return 0, fmt.Errorf("output %q, want %q", got, item.output)
	}
	return duration, nil
}

func writeCollection(state *collectionState) error {
	var sizes, timings []metric
	for index, item := range workloads {
		measurement := &state.measurements[index]
		timings = append(timings,
			durationMetric("compile/"+item.name, measurement.buildDurations),
			durationMetric("run/"+item.name, measurement.runDurations),
		)
		size, err := inspectExecutable(measurement.binary)
		if err != nil {
			return collectionError(state, "inspect", item.name, err)
		}
		sizes = append(sizes,
			byteMetric("binary/"+item.name+"/file", size.file),
			byteMetric("binary/"+item.name+"/text", size.text),
			byteMetric("binary/"+item.name+"/data", size.data),
			byteMetric("binary/"+item.name+"/bss", size.bss),
		)
	}
	if err := writeMetrics(filepath.Join(state.out, "size.json"), sizes); err != nil {
		return err
	}
	return writeMetrics(filepath.Join(state.out, "time.json"), timings)
}

func collectionError(state *collectionState, operation, workload string, err error) error {
	prefix := ""
	if state.name != "" {
		prefix = state.name + " "
	}
	return fmt.Errorf("%s%s %s: %w", prefix, operation, workload, err)
}

func parseInternalDuration(output string) (time.Duration, error) {
	value := strings.TrimSpace(output)
	nanoseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || nanoseconds < 0 {
		return 0, fmt.Errorf("invalid internal duration %q", value)
	}
	return time.Duration(nanoseconds), nil
}

func run(ctx context.Context, env []string, output io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func durationMetric(name string, values []time.Duration) metric {
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	middle := len(ordered) / 2
	median := float64(ordered[middle].Nanoseconds())
	if len(ordered)%2 == 0 {
		median = (float64(ordered[middle-1].Nanoseconds()) + median) / 2
	}
	return metric{
		Name:  name,
		Unit:  "ns",
		Value: median,
		Range: strconv.FormatInt(ordered[0].Nanoseconds(), 10) + ".." +
			strconv.FormatInt(ordered[len(ordered)-1].Nanoseconds(), 10),
		Extra: fmt.Sprintf("median of %d rotated runs", len(ordered)),
	}
}

func byteMetric(name string, value uint64) metric {
	return metric{Name: name, Unit: "bytes", Value: float64(value)}
}

func writeMetrics(path string, values []metric) error {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func executableFootprint(path string) (footprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return footprint{}, err
	}
	out := footprint{file: uint64(info.Size())}

	if f, err := elf.Open(path); err == nil {
		defer f.Close()
		addELFSections(&out, f.Sections)
		return out, nil
	}

	if f, err := macho.Open(path); err == nil {
		defer f.Close()
		addMachOSections(&out, f.Sections)
		return out, nil
	}

	return footprint{}, fmt.Errorf("unsupported executable format: %s", path)
}

func addELFSections(out *footprint, sections []*elf.Section) {
	for _, section := range sections {
		if section.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		switch {
		case section.Type == elf.SHT_NOBITS:
			out.bss += section.Size
		case section.Flags&elf.SHF_EXECINSTR != 0:
			out.text += section.Size
		default:
			out.data += section.Size
		}
	}
}

func addMachOSections(out *footprint, sections []*macho.Section) {
	for _, section := range sections {
		switch {
		case section.Seg == "__TEXT":
			out.text += section.Size
		case strings.HasPrefix(section.Seg, "__DATA") &&
			(section.Name == "__bss" || section.Name == "__common" || strings.HasSuffix(section.Name, "_bss")):
			out.bss += section.Size
		case strings.HasPrefix(section.Seg, "__DATA"):
			out.data += section.Size
		}
	}
}

func validateArtifact(dir string) error {
	sizeNames := make(map[string]string, len(workloads)*4)
	timeNames := make(map[string]string, len(workloads)*2)
	for _, item := range workloads {
		for _, part := range []string{"file", "text", "data", "bss"} {
			sizeNames["binary/"+item.name+"/"+part] = "bytes"
		}
		timeNames["compile/"+item.name] = "ns"
		timeNames["run/"+item.name] = "ns"
	}
	if err := validateMetrics(filepath.Join(dir, "size.json"), sizeNames); err != nil {
		return err
	}
	if err := validateMetrics(filepath.Join(dir, "time.json"), timeNames); err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(dir, "go.txt"))
	if err != nil {
		return err
	}
	defer f.Close()
	return validateGoBenchmarks(f)
}

var (
	metricRange = regexp.MustCompile(`^[0-9]+\.\.[0-9]+$`)
	metricExtra = regexp.MustCompile(`^[A-Za-z0-9 .,_:/()+-]+$`)
	cpuSuffix   = regexp.MustCompile(`-[0-9]+$`)
)

func validateMetrics(path string, expected map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	var values []metric
	if err := decoder.Decode(&values); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(values) != len(expected) {
		return fmt.Errorf("%s: got %d metrics, want %d", path, len(values), len(expected))
	}
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		unit, ok := expected[value.Name]
		if !ok {
			return fmt.Errorf("%s: unexpected metric %q", path, value.Name)
		}
		if seen[value.Name] {
			return fmt.Errorf("%s: duplicate metric %q", path, value.Name)
		}
		seen[value.Name] = true
		if value.Unit != unit {
			return fmt.Errorf("%s: metric %q has unit %q, want %q", path, value.Name, value.Unit, unit)
		}
		if math.IsNaN(value.Value) || math.IsInf(value.Value, 0) || value.Value < 0 {
			return fmt.Errorf("%s: metric %q has invalid value %v", path, value.Name, value.Value)
		}
		if value.Range != "" && !metricRange.MatchString(value.Range) {
			return fmt.Errorf("%s: metric %q has invalid range %q", path, value.Name, value.Range)
		}
		if value.Extra != "" && !metricExtra.MatchString(value.Extra) {
			return fmt.Errorf("%s: metric %q has invalid extra text %q", path, value.Name, value.Extra)
		}
	}
	return nil
}

func validateGoBenchmarks(r io.Reader) error {
	expected := make(map[string]int, len(expectedGoBenchmarks))
	for _, name := range expectedGoBenchmarks {
		expected[name] = 0
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		name := cpuSuffix.ReplaceAllString(fields[0], "")
		samples, ok := expected[name]
		if !ok {
			return fmt.Errorf("unexpected Go benchmark %q", name)
		}
		if samples >= goBenchmarkSamples {
			return fmt.Errorf("too many samples for Go benchmark %q", name)
		}
		if len(fields) < 4 || (len(fields)-2)%2 != 0 {
			return fmt.Errorf("malformed Go benchmark line %q", scanner.Text())
		}
		if _, err := strconv.ParseUint(fields[1], 10, 64); err != nil {
			return fmt.Errorf("benchmark %q has invalid iteration count: %w", name, err)
		}
		hasTime := false
		for i := 2; i < len(fields); i += 2 {
			value, err := strconv.ParseFloat(fields[i], 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return fmt.Errorf("benchmark %q has invalid value %q", name, fields[i])
			}
			switch fields[i+1] {
			case "ns/op":
				hasTime = true
			case "B/op", "allocs/op":
			default:
				return fmt.Errorf("benchmark %q has unexpected unit %q", name, fields[i+1])
			}
		}
		if !hasTime {
			return fmt.Errorf("benchmark %q has no ns/op result", name)
		}
		expected[name] = samples + 1
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	var missing []string
	for name, samples := range expected {
		if samples == 0 {
			missing = append(missing, name)
		}
	}
	slices.Sort(missing)
	if len(missing) != 0 {
		return fmt.Errorf("missing Go benchmarks: %s", strings.Join(missing, ", "))
	}
	for name, samples := range expected {
		if samples != goBenchmarkSamples {
			return fmt.Errorf("Go benchmark %q has %d samples, want %d", name, samples, goBenchmarkSamples)
		}
	}
	return nil
}
