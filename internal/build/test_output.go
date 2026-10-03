// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/internal/test2json"
)

// testOutputPolicy mirrors cmd/go's distinction between local-directory and
// package-list mode. Keep the original package count across feature groups;
// a single test binary in one group is not necessarily a single-package run.
type testOutputPolicy struct {
	stream bool
	direct bool
	show   bool
}

func newTestOutputPolicy(conf *Config, local bool, count int) *testOutputPolicy {
	var verbose, list, bench, fuzz, help bool
	for i := 0; i < len(conf.RunArgs); i++ {
		arg := conf.RunArgs[i]
		if arg == "--" {
			break
		}
		name, value, hasValue := strings.Cut(arg, "=")
		switch name {
		case "-test.v", "--test.v":
			verbose = !hasValue || value == "true" || value == "test2json"
		case "-test.list", "--test.list", "-test.bench", "--test.bench", "-test.fuzz", "--test.fuzz":
			if !hasValue && i+1 < len(conf.RunArgs) {
				i++
				value = conf.RunArgs[i]
			}
			switch name {
			case "-test.list", "--test.list":
				list = value != ""
			case "-test.bench", "--test.bench":
				bench = value != ""
			case "-test.fuzz", "--test.fuzz":
				fuzz = value != ""
			}
		case "-h", "--h", "-help", "--help":
			help = true
		}
	}
	show := verbose || list || help
	direct := local || bench || fuzz
	return &testOutputPolicy{
		stream: direct || show && (count == 1 || conf.BuildParallelism == 1) || conf.TestJSON,
		direct: direct,
		show:   show,
	}
}

func configureTestOutput(conf *Config, args []string, initial []*packages.Package) {
	if conf.testOutput != nil {
		return
	}
	paths := make(map[string]bool)
	for _, pkg := range initial {
		if pkg.ForTest == "" && !strings.HasSuffix(pkg.PkgPath, ".test") {
			paths[pkg.PkgPath] = true
		}
	}
	conf.testOutput = newTestOutputPolicy(conf, len(args) == 0, len(paths))
}

type lockedTestWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *lockedTestWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func testJSONArgs(args []string) []string {
	result := make([]string, 0, len(args)+1)
	for i, arg := range args {
		if arg == "--" {
			return append(append(result, "-test.v=test2json"), args[i:]...)
		}
		name, _, _ := strings.Cut(arg, "=")
		if name != "-test.v" && name != "--test.v" {
			result = append(result, arg)
		}
	}
	return append(result, "-test.v=test2json")
}

const noTestsMarker = "testing: warning: no tests to run"
const coverageMarker = "coverage: "

// testOutputMetadata retains only facts needed for the package summary. JSON
// runs do not cache test results, so keeping their complete output is wasteful.
type testOutputMetadata struct {
	noTests bool
	written bool
	last    byte
	tail    [len(noTestsMarker) - 1]byte
	tailLen int

	// A coverage line is bounded by the generated report and -coverpkg value,
	// not by the amount of test output or the length of arbitrary log lines.
	coverageLimit int
	coverage      []byte
	coverageDone  bool
}

func (m *testOutputMetadata) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	m.written, m.last = true, p[len(p)-1]
	var boundary [2 * (len(noTestsMarker) - 1)]byte
	n := copy(boundary[:], m.tail[:m.tailLen])
	n += copy(boundary[n:], p[:min(len(p), len(m.tail))])
	m.noTests = m.noTests || bytes.Contains(p, []byte(noTestsMarker)) || bytes.Contains(boundary[:n], []byte(noTestsMarker))
	m.captureCoverage(p, boundary[:n])
	if len(p) >= len(m.tail) {
		m.tailLen = copy(m.tail[:], p[len(p)-len(m.tail):])
	} else {
		m.tailLen = copy(m.tail[:], boundary[max(0, n-len(m.tail)):n])
	}
	return len(p), nil
}

func (m *testOutputMetadata) captureCoverage(p, boundary []byte) {
	if m.coverageLimit == 0 || m.coverageDone {
		return
	}
	if m.coverage == nil {
		if index := bytes.Index(boundary, []byte(coverageMarker)); index >= 0 && index < m.tailLen {
			m.coverage = append(m.coverage, coverageMarker...)
			p = p[index+len(coverageMarker)-m.tailLen:]
		} else if index := bytes.Index(p, []byte(coverageMarker)); index >= 0 {
			p = p[index:]
		} else {
			return
		}
	}
	line, _, ended := bytes.Cut(p, []byte{'\n'})
	if len(m.coverage)+len(line) > m.coverageLimit {
		// Arbitrary test logs must not turn this small summary tracker into
		// another unbounded buffer, even if they contain "coverage: ".
		m.coverage, m.coverageDone = nil, true
		return
	}
	m.coverage = append(m.coverage, line...)
	m.coverageDone = ended
}

// runTestProgram selects a writer before starting the child. Streaming output
// reaches the caller while the test is running; buffered output is returned to
// the coordinator for printing in package order.
func runTestProgram(program testProgram, conf *Config, count int, stdout, stderr io.Writer, run func(io.Writer) error) testProgramResult {
	policy := conf.testOutput
	if policy == nil {
		policy = newTestOutputPolicy(conf, false, count)
	}
	result := testProgramResult{program: program}
	var buffer bytes.Buffer
	var output io.Writer = &buffer
	if policy.stream {
		output = stdout
		if !policy.direct {
			output = io.MultiWriter(stdout, &buffer)
		}
	}
	var converter *test2json.Converter
	var metadata testOutputMetadata
	if conf.TestJSON {
		converter = test2json.NewConverter(stdout, program.pkgName, test2json.Timestamp)
		output = converter
		if !policy.direct && !program.coverage {
			output = io.MultiWriter(converter, &metadata)
		}
	}
	start := time.Now()
	result.err = run(output)
	elapsed := time.Since(start)
	var failure *runnerFailure
	if errors.As(result.err, &failure) {
		fmt.Fprintln(stderr, failure)
	}
	if !program.coverage {
		norun := ""
		if metadata.noTests || bytes.Contains(buffer.Bytes(), []byte(noTestsMarker)) {
			norun = " [no tests to run]"
		}
		if result.err == nil && !policy.show && !conf.TestJSON && !policy.stream {
			buffer.Reset()
		}
		if metadata.written && metadata.last != '\n' || buffer.Len() != 0 && buffer.Bytes()[buffer.Len()-1] != '\n' {
			fmt.Fprintln(output)
		}
		if result.err != nil {
			prefix := ""
			if conf.TestJSON {
				prefix = "\x16"
			}
			fmt.Fprintf(output, "%sFAIL\t%s\t%.3fs\n", prefix, program.pkgName, elapsed.Seconds())
		} else {
			fmt.Fprintf(output, "ok  \t%s\t%.3fs%s\n", program.pkgName, elapsed.Seconds(), norun)
		}
	}
	if converter != nil {
		converter.Exited(result.err)
		result.err = errors.Join(result.err, converter.Close())
	} else if !policy.stream {
		result.output = buffer.Bytes()
	}
	return result
}

// report accepts completions in any order, but prints buffered package records
// in selection order, as cmd/go's test-print actions do. Skipped/build-failed
// roots must also advance the queue so they cannot hold later results forever.
func testResultReporter(count int, stdout io.Writer) func(int, *testProgramResult) {
	results := make([]*testProgramResult, count)
	ready := make([]bool, count)
	next := 0
	return func(index int, result *testProgramResult) {
		results[index], ready[index] = result, true
		for next < count && ready[next] {
			if result := results[next]; result != nil && len(result.output) != 0 {
				stdout.Write(result.output)
			}
			results[next] = nil
			next++
		}
	}
}
