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

// Package gdb launches GDB with LLGo runtime presentation support.
package gdb

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/debugabi"
	"github.com/xgo-dev/llgo/cmd/compiler/mockable"
	"github.com/xgo-dev/llgo/cmd/internal/base"
)

const minimumGDBVersion = 12
const debuggerSchemaFilename = "llgo_debugger_schema_v1.json"

var (
	//go:embed llgo_plugin.py
	pluginSource []byte

	gdbVersionPattern = regexp.MustCompile(`(?im)^GNU gdb[^\n]*?([0-9]+)(?:\.[0-9]+)`)
)

// Cmd is the direct GDB launcher; orchestration is supplied by llgo debug.
var Cmd = &base.Command{
	UsageLine: "llgo gdb [-gdb path] [--] executable [gdb arguments...]",
	Short:     "Debug an LLGo executable with GDB",
}

var gdbPath string

func init() {
	Cmd.Flag.StringVar(&gdbPath, "gdb", "", "path to GNU GDB with Python support (default $LLGO_GDB or auto-detect)")
	Cmd.Run = func(cmd *base.Command, args []string) {
		if err := cmd.Flag.Parse(args); err != nil {
			mockable.Exit(2)
			return
		}
		if len(cmd.Flag.Args()) == 0 {
			fmt.Fprintln(os.Stderr, "llgo gdb: no executable specified")
			mockable.Exit(2)
			return
		}
		if err := Run(gdbPath, nil, cmd.Flag.Args(), os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			mockable.Exit(1)
		}
	}
}

// Run starts GDB with the LLGo adapter. Candidates are target-specific GDB
// commands in preference order; the configured path and LLGO_GDB take
// precedence over them.
func Run(configuredPath string, candidates, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	path, err := findGDB(configuredPath, candidates)
	if err != nil {
		return err
	}

	pluginDir, err := os.MkdirTemp("", "llgo-gdb-")
	if err != nil {
		return fmt.Errorf("llgo gdb: create GDB plugin directory: %w", err)
	}
	defer os.RemoveAll(pluginDir)

	pluginPath := filepath.Join(pluginDir, "llgo_plugin.py")
	if err := os.WriteFile(pluginPath, pluginSource, 0600); err != nil {
		return fmt.Errorf("llgo gdb: write GDB plugin: %w", err)
	}
	schemaPath := filepath.Join(pluginDir, debuggerSchemaFilename)
	if err := os.WriteFile(schemaPath, debugabi.SchemaV1(), 0600); err != nil {
		return fmt.Errorf("llgo gdb: write debugger schema: %w", err)
	}

	gdbArgs := make([]string, 0, len(args)+2)
	gdbArgs = append(gdbArgs, "-ex", gdbSourceCommand(pluginPath))
	gdbArgs = append(gdbArgs, args...)
	command := exec.Command(path, gdbArgs...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("llgo gdb: GDB: %w", err)
	}
	return nil
}

func findGDB(configuredPath string, candidates []string) (string, error) {
	return findGDBFrom(configuredPath, os.Getenv("LLGO_GDB"), candidates, []string{
		"gdb-multiarch",
		"gdb",
	})
}

func findGDBFrom(configuredPath, environmentPath string, candidates, fallbacks []string) (string, error) {
	if configuredPath != "" {
		return validateGDB(configuredPath)
	}
	if environmentPath != "" {
		return validateGDB(environmentPath)
	}

	seen := make(map[string]bool)
	for _, candidate := range append(append([]string(nil), candidates...), fallbacks...) {
		path, err := exec.LookPath(candidate)
		if err != nil || seen[path] {
			continue
		}
		seen[path] = true
		if path, err = validateResolvedGDB(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("llgo gdb: GNU GDB %d or newer with Python support not found; install GDB or set LLGO_GDB", minimumGDBVersion)
}

func validateGDB(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("llgo gdb: find GDB %q: %w", name, err)
	}
	return validateResolvedGDB(path)
}

func validateResolvedGDB(path string) (string, error) {
	output, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llgo gdb: query GDB %q version: %w", path, err)
	}
	match := gdbVersionPattern.FindStringSubmatch(string(output))
	if len(match) != 2 {
		return "", fmt.Errorf("llgo gdb: cannot parse GDB version from %q", strings.TrimSpace(string(output)))
	}
	major, err := strconv.Atoi(match[1])
	if err != nil || major < minimumGDBVersion {
		return "", fmt.Errorf("llgo gdb: %q is GNU GDB %s; version %d or newer is required", path, match[1], minimumGDBVersion)
	}
	if output, err = exec.Command(path, "--batch", "--nx", "-ex", "python import json, re").CombinedOutput(); err != nil {
		return "", fmt.Errorf("llgo gdb: GDB %q requires Python support: %s", path, strings.TrimSpace(string(output)))
	}
	return path, nil
}

func gdbSourceCommand(path string) string {
	path = filepath.ToSlash(path)
	path = strings.NewReplacer(`\`, `\\`, ` `, `\ `, `"`, `\"`).Replace(path)
	return `source ` + path
}
