// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.
// See LICENSE for details.

package build

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrTestFailed reports tests whose package results have already been printed.
// Return it instead of exiting inside Build so temporary coverage files are
// cleaned up before the command chooses its exit status and final summary.
var ErrTestFailed = errors.New("test failed")

func coverageTestFailure(errs []error) error {
	if err := errors.Join(errs...); err != nil {
		return err
	}
	return ErrTestFailed
}

// Packages without tests still build, and contribute zero counts through the
// cover tool's static metadata. Go's covdata tool owns their report format.
func (c *coverageBuild) reportNoTests(conf *Config) error {
	if conf.CompileOnly {
		return nil
	}
	for _, p := range c.noTests {
		var report bytes.Buffer
		meta := c.metaPaths[p.PkgPath]
		// Instrumentation records only existing, nonempty metadata files.
		if meta != "" {
			dir := filepath.Dir(meta)
			cmd := c.commands.configure(exec.Command(c.goCommand, "tool", "covdata", "percent", "-i="+dir))
			cmd.Stdout = &report
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
			if c.options.Profile != "" {
				profile := filepath.Join(dir, "cover.out")
				cmd = c.commands.configure(exec.Command(c.goCommand, "tool", "covdata", "textfmt", "-i="+dir, "-o="+profile))
				if output, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("go tool covdata: %w\n%s", err, output)
				}
				if err := c.merge(profile); err != nil {
					return err
				}
			}
		} else {
			fmt.Fprintf(&report, "?   \t%s\t[no test files]\n", p.PkgPath)
		}
		if conf.TestJSON {
			cmd := c.commands.configure(exec.Command(c.goCommand, "tool", "test2json", "-t", "-p", p.PkgPath))
			cmd.Stdin = &report
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		} else if _, err := os.Stdout.Write(report.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (c *coverageBuild) runArgs(args []string) ([]string, string, error) {
	dir, err := os.MkdirTemp(c.dir, "run-")
	if err != nil {
		return nil, "", err
	}
	if len(c.manifest) != 0 {
		if err := os.WriteFile(filepath.Join(dir, "metafiles.txt"), c.manifest, 0600); err != nil {
			return nil, "", err
		}
	}
	args = append([]string{"-test.gocoverdir=" + dir}, args...)
	profile := ""
	if c.options.Profile != "" {
		profile = filepath.Join(dir, "cover.out")
		args = append([]string{"-test.coverprofile=" + profile}, args...)
	}
	return args, profile, nil
}

func (c *coverageBuild) merge(profile string) error {
	if profile == "" {
		return nil
	}
	f, err := os.Open(profile)
	if os.IsNotExist(err) {
		// A crashing test may not emit a profile.
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	header := "mode: " + c.options.Mode + "\n"
	buf := make([]byte, len(header))
	n, err := io.ReadFull(f, buf)
	if n == 0 && err == io.EOF {
		return nil
	}
	if err != nil || string(buf) != header {
		return fmt.Errorf("test wrote malformed coverage profile %s", profile)
	}
	c.mergeMu.Lock()
	defer c.mergeMu.Unlock()
	out, err := os.OpenFile(c.options.Profile, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, f)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func runCoveredTest(commands commandEnv, program testProgram, conf *Config, stdout, stderr io.Writer) error {
	c := conf.coverage
	args, profile, err := c.runArgs(conf.RunArgs)
	if err != nil {
		return err
	}
	copyConf := *conf
	copyConf.coverage = nil
	if conf.TestJSON {
		args = testJSONArgs(args)
	}
	copyConf.RunArgs = args
	if conf.PrintCommands {
		// Driver diagnostics belong outside the test's buffered/JSON output.
		fmt.Fprintf(stderr, "%s %s\n", program.app, strings.Join(args, " "))
		copyConf.PrintCommands = false
	}
	var output bytes.Buffer
	var testOutput io.Writer = &output
	policy := conf.testOutput
	if policy == nil {
		policy = newTestOutputPolicy(conf, c.local, 1)
	}
	stream := policy.stream
	var metadata testOutputMetadata
	if stream {
		testOutput = stdout
		if !policy.direct {
			if conf.TestJSON {
				metadata.coverageLimit = 128 + len(c.options.Packages)
				testOutput = io.MultiWriter(stdout, &metadata)
			} else {
				testOutput = io.MultiWriter(stdout, &output)
			}
		}
	}
	start := time.Now()
	err = runNativeTest(commands, program, &copyConf, testOutput, testOutput)
	elapsed := time.Since(start)
	if metadata.written && metadata.last != '\n' || output.Len() != 0 && output.Bytes()[output.Len()-1] != '\n' {
		fmt.Fprintln(testOutput)
	}
	mergeErr := c.merge(profile)
	err = errors.Join(err, mergeErr)
	var report bytes.Buffer
	if !stream && (policy.show || err != nil) {
		report.Write(output.Bytes())
	}
	if mergeErr != nil {
		fmt.Fprintln(&report, mergeErr)
	}
	if err != nil {
		prefix := ""
		if conf.TestJSON {
			prefix = "\x16"
		}
		fmt.Fprintf(&report, "%sFAIL\t%s\t%.3fs\n", prefix, program.pkgName, elapsed.Seconds())
	} else {
		suffix := ""
		if !c.local && len(metadata.coverage) != 0 {
			suffix = "\t" + string(metadata.coverage)
		}
		for _, line := range strings.Split(output.String(), "\n") {
			if index := strings.Index(line, "coverage: "); !c.local && index >= 0 {
				// Go 1.20 prefixes this line with the package path. Like
				// cmd/go, retain only the coverage result in the ok record.
				suffix = "\t" + line[index:]
				break
			}
		}
		if metadata.noTests || bytes.Contains(output.Bytes(), []byte(noTestsMarker)) {
			suffix += " [no tests to run]"
		}
		fmt.Fprintf(&report, "ok  \t%s\t%.3fs%s\n", program.pkgName, elapsed.Seconds(), suffix)
	}
	_, writeErr := stdout.Write(report.Bytes())
	err = errors.Join(err, writeErr)
	return err
}
