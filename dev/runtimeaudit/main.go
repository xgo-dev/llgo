// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

// runtimeaudit emits the runtime source ownership report for an application
// toolchain and target. Run from an LLGo development environment.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/env"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	var conf build.RuntimeAuditConfig
	flags := flag.NewFlagSet("runtimeaudit", flag.ContinueOnError)
	flags.StringVar(&conf.Dir, "dir", ".", "application directory for Go toolchain selection")
	flags.StringVar(&conf.RuntimeDir, "runtime", "", "LLGo runtime source directory")
	flags.StringVar(&conf.GOOS, "goos", runtime.GOOS, "target GOOS")
	flags.StringVar(&conf.GOARCH, "goarch", runtime.GOARCH, "target GOARCH")
	tags := flags.String("tags", "", "target build tags, including hosted Wasm profile tags")
	out := flags.String("o", "", "JSON report file (default stdout)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if conf.RuntimeDir == "" {
		conf.RuntimeDir = env.LLGoRuntimeDir()
	}
	conf.Tags = strings.FieldsFunc(*tags, func(r rune) bool { return r == ',' || r == ' ' })
	report, err := build.AuditRuntimeSources(conf)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *out == "" {
		_, err = stdout.Write(data)
	} else {
		err = os.WriteFile(*out, data, 0644)
	}
	if err != nil {
		return err
	}
	if len(report.Missing) != 0 {
		return fmt.Errorf("%d unresolved runtime references; see missing in the report", len(report.Missing))
	}
	if len(report.DynamicCompilerUses) != 0 {
		return fmt.Errorf("%d unbounded compiler runtime lookups; see dynamic_compiler_uses in the report", len(report.DynamicCompilerUses))
	}
	if len(report.UnresolvedBoundaries) != 0 {
		return fmt.Errorf("%d runtime boundaries without a provider; see unresolved_boundaries in the report", len(report.UnresolvedBoundaries))
	}
	return nil
}
