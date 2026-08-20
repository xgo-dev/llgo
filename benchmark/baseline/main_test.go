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
	"bytes"
	"context"
	"debug/elf"
	"debug/macho"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDurationMetric(t *testing.T) {
	values := []time.Duration{9, 3, 6}
	got := durationMetric("compile/test", values)
	if got.Name != "compile/test" || got.Unit != "ns" || got.Value != 6 {
		t.Fatalf("durationMetric = %+v", got)
	}
	if got.Range != "3..9" || got.Extra != "median of 3 rotated runs" {
		t.Fatalf("duration metadata = %+v", got)
	}
	if !slices.Equal(values, []time.Duration{9, 3, 6}) {
		t.Fatalf("durationMetric mutated input: %v", values)
	}

	even := durationMetric("compile/even", []time.Duration{8, 2})
	if even.Value != 5 || even.Range != "2..8" {
		t.Fatalf("even durationMetric = %+v", even)
	}
}

func TestParseInternalDuration(t *testing.T) {
	got, err := parseInternalDuration(" 123\n")
	if err != nil || got != 123*time.Nanosecond {
		t.Fatalf("parseInternalDuration = %v, %v", got, err)
	}
	for _, input := range []string{"", "not-a-duration", "-1"} {
		if _, err := parseInternalDuration(input); err == nil {
			t.Fatalf("parseInternalDuration(%q) unexpectedly succeeded", input)
		}
	}
}

func TestWriteAndValidateMetrics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")
	values := []metric{
		{Name: "one", Unit: "bytes", Value: 1},
		{Name: "two", Unit: "bytes", Value: 2, Range: "1..3", Extra: "median of 3 consecutive runs"},
	}
	if err := writeMetrics(path, values); err != nil {
		t.Fatal(err)
	}
	if err := validateMetrics(path, map[string]string{"one": "bytes", "two": "bytes"}); err != nil {
		t.Fatal(err)
	}

	var decoded []metric
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[1].Range != "1..3" {
		t.Fatalf("decoded metrics = %+v", decoded)
	}
}

func TestExportBenchmarks(t *testing.T) {
	dir := t.TempDir()
	writeValidArtifact(t, dir)
	output := filepath.Join(dir, "benchmark.txt")
	if err := exportBenchmarks(dir, output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"pkg: github.com/xgo-dev/llgo/benchmark/baseline",
		"Unit file-bytes better=lower assume=exact",
		"Unit build-ns better=lower",
		"BenchmarkProgram/cprintf 1 1 file-bytes 1 text-bytes 1 data-bytes 1 bss-bytes 1 build-ns 1 run-ns",
		"BenchmarkProgram/memprofile-no-consumer 1 1 file-bytes 1 text-bytes 1 data-bytes 1 bss-bytes 1 build-ns 1 run-ns",
		"BenchmarkRuntimeGetG-1 100 12.5 ns/op",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("export does not contain %q:\n%s", want, text)
		}
	}
}

func TestExportBenchmarksRejectsInvalidInput(t *testing.T) {
	if err := exportBenchmarks(t.TempDir(), ""); err == nil ||
		!strings.Contains(err.Error(), "benchmark-output") {
		t.Fatalf("exportBenchmarks error = %v", err)
	}
}

func TestWriteMetricsRejectsDirectory(t *testing.T) {
	if err := writeMetrics(t.TempDir(), []metric{{Name: "one", Unit: "bytes", Value: 1}}); err == nil {
		t.Fatal("writeMetrics unexpectedly accepted a directory")
	}
	path := filepath.Join(t.TempDir(), "nan.json")
	if err := writeMetrics(path, []metric{{Name: "one", Unit: "bytes", Value: math.NaN()}}); err == nil {
		t.Fatal("writeMetrics unexpectedly accepted NaN")
	}
}

func TestValidateMetricsRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"unknown field", `[{"name":"one","unit":"bytes","value":1,"bad":1}]`, "unknown field"},
		{"wrong count", `[]`, "got 0 metrics"},
		{"unexpected name", `[{"name":"two","unit":"bytes","value":1}]`, `unexpected metric "two"`},
		{"wrong unit", `[{"name":"one","unit":"ns","value":1}]`, `unit "ns"`},
		{"negative", `[{"name":"one","unit":"bytes","value":-1}]`, "invalid value"},
		{"bad range", `[{"name":"one","unit":"bytes","value":1,"range":"bad"}]`, "invalid range"},
		{"bad extra", `[{"name":"one","unit":"bytes","value":1,"extra":"<script>"}]`, "invalid extra"},
		{"duplicate", `[{"name":"one","unit":"bytes","value":1},{"name":"one","unit":"bytes","value":2}]`, "duplicate metric"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metrics.json")
			if err := os.WriteFile(path, []byte(tt.data), 0o644); err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{"one": "bytes"}
			if tt.name == "duplicate" {
				expected["two"] = "bytes"
			}
			err := validateMetrics(path, expected)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateMetrics error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateGoBenchmarks(t *testing.T) {
	var input strings.Builder
	for _, name := range expectedGoBenchmarks {
		for range goBenchmarkSamples {
			input.WriteString(name)
			input.WriteString("-1 100 12.5 ns/op\n")
		}
	}
	if err := validateGoBenchmarks(strings.NewReader(input.String())); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGoBenchmarksRejectsInvalidData(t *testing.T) {
	valid := func(skip string) string {
		var input strings.Builder
		for _, name := range expectedGoBenchmarks {
			if name != skip {
				for range goBenchmarkSamples {
					input.WriteString(name + "-1 100 12.5 ns/op\n")
				}
			}
		}
		return input.String()
	}
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"missing", valid(expectedGoBenchmarks[0]), "missing Go benchmarks"},
		{"unknown", valid("") + "BenchmarkUnknown-1 1 1 ns/op\n", `unexpected Go benchmark "BenchmarkUnknown"`},
		{"too many samples", valid("") + expectedGoBenchmarks[0] + "-1 1 1 ns/op\n", "too many samples"},
		{"too few samples", strings.Replace(valid(""), expectedGoBenchmarks[0]+"-1 100 12.5 ns/op\n", "", 1), fmt.Sprintf("has %d samples, want %d", goBenchmarkSamples-1, goBenchmarkSamples)},
		{"malformed", strings.Replace(valid(""), " 100 12.5 ns/op", " bad", 1), "malformed Go benchmark"},
		{"iterations", strings.Replace(valid(""), " 100 ", " bad ", 1), "invalid iteration count"},
		{"value", strings.Replace(valid(""), "12.5", "bad", 1), "invalid value"},
		{"unit", strings.Replace(valid(""), "ns/op", "widgets", 1), "unexpected unit"},
		{"no time", strings.Replace(valid(""), "12.5 ns/op", "12 B/op", 1), "has no ns/op"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGoBenchmarks(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateGoBenchmarks error = %v, want %q", err, tt.want)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestValidateGoBenchmarksReportsReadError(t *testing.T) {
	err := validateGoBenchmarks(failingReader{})
	if err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("validateGoBenchmarks error = %v", err)
	}
}

func TestExecutableFootprint(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got, err := executableFootprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.file == 0 || got.text == 0 {
		t.Fatalf("executable footprint = %+v", got)
	}
}

func TestExecutableFootprintErrors(t *testing.T) {
	if _, err := executableFootprint(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("executableFootprint unexpectedly accepted a missing file")
	}
	path := filepath.Join(t.TempDir(), "text")
	if err := os.WriteFile(path, []byte("not an executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := executableFootprint(path); err == nil || !strings.Contains(err.Error(), "unsupported executable format") {
		t.Fatalf("executableFootprint error = %v", err)
	}
}

func TestAddELFSections(t *testing.T) {
	var got footprint
	addELFSections(&got, []*elf.Section{
		{SectionHeader: elf.SectionHeader{Size: 99}},
		{SectionHeader: elf.SectionHeader{Flags: elf.SHF_ALLOC | elf.SHF_EXECINSTR, Size: 10}},
		{SectionHeader: elf.SectionHeader{Flags: elf.SHF_ALLOC, Size: 20}},
		{SectionHeader: elf.SectionHeader{Flags: elf.SHF_ALLOC, Type: elf.SHT_NOBITS, Size: 30}},
	})
	if got.text != 10 || got.data != 20 || got.bss != 30 {
		t.Fatalf("ELF footprint = %+v", got)
	}
}

func TestAddMachOSections(t *testing.T) {
	var got footprint
	addMachOSections(&got, []*macho.Section{
		{SectionHeader: macho.SectionHeader{Seg: "__TEXT", Name: "__text", Size: 10}},
		{SectionHeader: macho.SectionHeader{Seg: "__DATA", Name: "__data", Size: 20}},
		{SectionHeader: macho.SectionHeader{Seg: "__DATA", Name: "__bss", Size: 30}},
		{SectionHeader: macho.SectionHeader{Seg: "__DATA_CONST", Name: "__const", Size: 40}},
		{SectionHeader: macho.SectionHeader{Seg: "__DATA_DIRTY", Name: "__thread_bss", Size: 50}},
		{SectionHeader: macho.SectionHeader{Seg: "__DWARF", Name: "__debug_info", Size: 99}},
	})
	if got.text != 10 || got.data != 60 || got.bss != 80 {
		t.Fatalf("Mach-O footprint = %+v", got)
	}
}

func TestCollect(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake compiler uses a POSIX shell")
	}
	root := t.TempDir()
	fakeLLGo := writeFakeCompiler(t, root, "printf 'Hello, world\\n'")

	oldInspect := inspectExecutable
	inspectExecutable = func(path string) (footprint, error) {
		info, err := os.Stat(path)
		if err != nil {
			return footprint{}, err
		}
		return footprint{file: uint64(info.Size()), text: 10, data: 2, bss: 1}, nil
	}
	t.Cleanup(func() {
		inspectExecutable = oldInspect
	})

	out := filepath.Join(root, "out")
	if err := collect(context.Background(), root, fakeLLGo, out, 2, 2); err != nil {
		t.Fatal(err)
	}
	goText := makeGoBenchmarkText()
	if err := os.WriteFile(filepath.Join(out, "go.txt"), []byte(goText), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateArtifact(out); err != nil {
		t.Fatal(err)
	}
}

func TestCollectPaired(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake compiler uses a POSIX shell")
	}
	root := t.TempDir()
	baseRoot := filepath.Join(root, "base")
	currentRoot := filepath.Join(root, "current")
	if err := os.MkdirAll(baseRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(currentRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	baseLLGo := writeFakeCompiler(t, baseRoot, "printf 'Hello, world\\n'")
	currentLLGo := writeFakeCompiler(t, currentRoot, "printf 'Hello, world\\n'")

	oldInspect := inspectExecutable
	inspectExecutable = func(path string) (footprint, error) {
		info, err := os.Stat(path)
		if err != nil {
			return footprint{}, err
		}
		return footprint{file: uint64(info.Size()), text: 10, data: 2, bss: 1}, nil
	}
	t.Cleanup(func() {
		inspectExecutable = oldInspect
	})

	baseOut := filepath.Join(root, "base-out")
	currentOut := filepath.Join(root, "current-out")
	err := collectPaired(
		context.Background(),
		collectionSpec{name: "base", root: baseRoot, harnessRoot: root, llgo: baseLLGo, out: baseOut},
		collectionSpec{name: "current", root: currentRoot, harnessRoot: root, llgo: currentLLGo, out: currentOut},
		2,
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{baseOut, currentOut} {
		if err := os.WriteFile(filepath.Join(out, "go.txt"), []byte(makeGoBenchmarkText()), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := validateArtifact(out); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCollectionStateIndexAlternatesEachWorkload(t *testing.T) {
	for workloadIndex := range workloads {
		first := collectionStateIndex(0, workloadIndex, 0, 2)
		if second := collectionStateIndex(1, workloadIndex, 0, 2); second == first {
			t.Fatalf("workload %d keeps state %d first across rounds", workloadIndex, first)
		}
		if peer := collectionStateIndex(0, workloadIndex, 1, 2); peer == first {
			t.Fatalf("workload %d state order repeats %d within a round", workloadIndex, first)
		}
	}
}

func TestCollectRejectsInvalidRuns(t *testing.T) {
	err := collect(context.Background(), ".", "llgo", t.TempDir(), 0, 1)
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("collect error = %v", err)
	}
}

func TestCollectReportsStages(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake compiler uses a POSIX shell")
	}
	tests := []struct {
		name     string
		compiler func(*testing.T, string) string
		inspect  func(string) (footprint, error)
		want     string
	}{
		{
			name: "build",
			compiler: func(t *testing.T, root string) string {
				return writeScript(t, filepath.Join(root, "llgo"), "#!/bin/sh\nexit 1\n")
			},
			inspect: func(string) (footprint, error) { return footprint{}, nil },
			want:    "build cprintf",
		},
		{
			name: "inspect",
			compiler: func(t *testing.T, root string) string {
				return writeFakeCompiler(t, root, "printf 'Hello, world\\n'")
			},
			inspect: func(string) (footprint, error) { return footprint{}, errors.New("inspect failed") },
			want:    "inspect cprintf",
		},
		{
			name: "execute",
			compiler: func(t *testing.T, root string) string {
				return writeFakeCompiler(t, root, "exit 1")
			},
			inspect: func(string) (footprint, error) { return footprint{file: 1}, nil },
			want:    "execute cprintf",
		},
		{
			name: "output",
			compiler: func(t *testing.T, root string) string {
				return writeFakeCompiler(t, root, "printf 'wrong\\n'")
			},
			inspect: func(string) (footprint, error) { return footprint{file: 1}, nil },
			want:    `output "wrong\n"`,
		},
		{
			name: "timed execute",
			compiler: func(t *testing.T, root string) string {
				return writeFakeCompiler(t, root, `
if [ -e "$0.state" ]; then
  exit 1
fi
touch "$0.state"
printf 'Hello, world\n'
`)
			},
			inspect: func(string) (footprint, error) { return footprint{file: 1}, nil },
			want:    "execute cprintf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			compiler := tt.compiler(t, root)
			oldInspect := inspectExecutable
			inspectExecutable = tt.inspect
			t.Cleanup(func() {
				inspectExecutable = oldInspect
			})
			err := collect(context.Background(), root, compiler, filepath.Join(root, "out"), 1, 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("collect error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCollectReportsResultWriteError(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("fake compiler uses a POSIX shell")
	}
	root := t.TempDir()
	compiler := writeFakeCompiler(t, root, "printf 'Hello, world\\n'")
	out := filepath.Join(root, "out")
	oldInspect := inspectExecutable
	inspectExecutable = func(string) (footprint, error) {
		if err := os.MkdirAll(filepath.Join(out, "size.json"), 0o755); err != nil {
			return footprint{}, err
		}
		return footprint{file: 1, text: 1}, nil
	}
	t.Cleanup(func() {
		inspectExecutable = oldInspect
	})
	err := collect(context.Background(), root, compiler, out, 1, 1)
	if err == nil || !strings.Contains(err.Error(), "size.json") {
		t.Fatalf("collect error = %v", err)
	}
}

func TestRunReportsCommand(t *testing.T) {
	var output bytes.Buffer
	err := run(context.Background(), os.Environ(), &output, "definitely-not-an-llgo-command")
	if err == nil || !strings.Contains(err.Error(), "definitely-not-an-llgo-command") {
		t.Fatalf("run error = %v", err)
	}
}

func TestValidateArtifactReportsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := validateArtifact(dir); err == nil || !strings.Contains(err.Error(), "size.json") {
		t.Fatalf("missing size error = %v", err)
	}

	var sizes, timings []metric
	for _, item := range workloads {
		for _, part := range []string{"file", "text", "data", "bss"} {
			sizes = append(sizes, metric{Name: "binary/" + item.name + "/" + part, Unit: "bytes", Value: 1})
		}
		timings = append(timings,
			metric{Name: "compile/" + item.name, Unit: "ns", Value: 1},
			metric{Name: "run/" + item.name, Unit: "ns", Value: 1},
		)
	}
	if err := writeMetrics(filepath.Join(dir, "size.json"), sizes); err != nil {
		t.Fatal(err)
	}
	if err := validateArtifact(dir); err == nil || !strings.Contains(err.Error(), "time.json") {
		t.Fatalf("missing time error = %v", err)
	}
	if err := writeMetrics(filepath.Join(dir, "time.json"), timings); err != nil {
		t.Fatal(err)
	}
	if err := validateArtifact(dir); err == nil || !strings.Contains(err.Error(), "go.txt") {
		t.Fatalf("missing go benchmark error = %v", err)
	}
}

func TestRunCLI(t *testing.T) {
	if err := runCLI(context.Background(), []string{"-build-runs=0"}); err == nil ||
		!strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("collect CLI error = %v", err)
	}
	if err := runCLI(context.Background(), []string{"-mode=unknown"}); err == nil ||
		!strings.Contains(err.Error(), "unknown mode") {
		t.Fatalf("unknown CLI error = %v", err)
	}
	if err := runCLI(context.Background(), []string{"-mode=collect-paired"}); err == nil ||
		!strings.Contains(err.Error(), "requires base-root") {
		t.Fatalf("collect-paired CLI error = %v", err)
	}
	if err := runCLI(context.Background(), []string{"-not-a-flag"}); err == nil {
		t.Fatal("runCLI unexpectedly accepted an unknown flag")
	}

	dir := t.TempDir()
	writeValidArtifact(t, dir)
	if err := runCLI(context.Background(), []string{"-mode=validate", "-out", dir}); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "benchmark.txt")
	if err := runCLI(context.Background(), []string{
		"-mode=export",
		"-out", dir,
		"-benchmark-output", output,
	}); err != nil {
		t.Fatal(err)
	}
}

func writeFakeCompiler(t *testing.T, root, program string) string {
	t.Helper()
	script := `#!/bin/sh
set -eu
out=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
case "$out" in
  *memprofile-*)
    cat > "$out" <<'LLGO_BENCH_PROGRAM'
#!/bin/sh
printf '1\n'
LLGO_BENCH_PROGRAM
    ;;
  *)
    cat > "$out" <<'LLGO_BENCH_PROGRAM'
#!/bin/sh
` + program + `
LLGO_BENCH_PROGRAM
    ;;
esac
chmod +x "$out"
`
	return writeScript(t, filepath.Join(root, "llgo"), script)
}

func writeScript(t *testing.T, path, script string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeValidArtifact(t *testing.T, dir string) {
	t.Helper()
	var sizes, timings []metric
	for _, item := range workloads {
		for _, part := range []string{"file", "text", "data", "bss"} {
			sizes = append(sizes, metric{Name: "binary/" + item.name + "/" + part, Unit: "bytes", Value: 1})
		}
		timings = append(timings,
			metric{Name: "compile/" + item.name, Unit: "ns", Value: 1},
			metric{Name: "run/" + item.name, Unit: "ns", Value: 1},
		)
	}
	if err := writeMetrics(filepath.Join(dir, "size.json"), sizes); err != nil {
		t.Fatal(err)
	}
	if err := writeMetrics(filepath.Join(dir, "time.json"), timings); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.txt"), []byte(makeGoBenchmarkText()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeGoBenchmarkText() string {
	var input strings.Builder
	for _, name := range expectedGoBenchmarks {
		for range goBenchmarkSamples {
			input.WriteString(name + "-1 100 12.5 ns/op\n")
		}
	}
	return input.String()
}

var _ io.Reader = failingReader{}
