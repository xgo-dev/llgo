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

// Package lldb implements the "llgo lldb" command.
package lldb

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/internal/mockable"
)

const minimumUpstreamLLDBVersion = 18

var (
	//go:embed llgo_plugin.py
	pluginSource []byte

	upstreamLLDBVersionPattern = regexp.MustCompile(`(?i)\blldb\s+version\s+([0-9]+)`)
	appleLLDBVersionPattern    = regexp.MustCompile(`(?i)\blldb-([0-9]+)`)
	lldbPath                   string
)

type lldbVersion struct {
	major int
	apple bool
}

// Cmd is the llgo lldb command.
var Cmd = &base.Command{
	UsageLine: "llgo lldb [-lldb path] [--] executable [lldb arguments...]",
	Short:     "Debug an LLGo executable with LLDB",
}

func init() {
	Cmd.Run = runCmd
	Cmd.Flag.StringVar(&lldbPath, "lldb", "", "path to upstream LLDB 18 or newer, or Apple LLDB (default $LLGO_LLDB or auto-detect)")
}

func runCmd(cmd *base.Command, args []string) {
	if err := cmd.Flag.Parse(args); err != nil {
		mockable.Exit(2)
		return
	}
	if err := run(lldbPath, cmd.Flag.Args(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}

func run(configuredPath string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("llgo lldb: no executable specified")
	}

	path, err := findLLDB(configuredPath)
	if err != nil {
		return err
	}

	pluginDir, err := os.MkdirTemp("", "llgo-lldb-")
	if err != nil {
		return fmt.Errorf("llgo lldb: create plugin directory: %w", err)
	}
	defer os.RemoveAll(pluginDir)

	pluginPath := filepath.Join(pluginDir, "llgo_plugin.py")
	if err := os.WriteFile(pluginPath, pluginSource, 0600); err != nil {
		return fmt.Errorf("llgo lldb: write plugin: %w", err)
	}

	lldbArgs := make([]string, 0, len(args)+2)
	lldbArgs = append(lldbArgs, "-O", lldbImportCommand(pluginPath))
	lldbArgs = append(lldbArgs, args...)

	command := exec.Command(path, lldbArgs...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("llgo lldb: %w", err)
	}
	return nil
}

func findLLDB(configuredPath string) (string, error) {
	return findLLDBFrom(configuredPath, os.Getenv("LLGO_LLDB"), []string{
		"/opt/homebrew/bin/lldb",
		"/usr/local/bin/lldb",
		"/usr/bin/lldb",
		"lldb",
	})
}

func findLLDBFrom(configuredPath, environmentPath string, candidates []string) (string, error) {
	if configuredPath != "" {
		return validateLLDB(configuredPath)
	}
	if environmentPath != "" {
		return validateLLDB(environmentPath)
	}

	seen := make(map[string]bool)
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate)
		if err != nil || seen[path] {
			continue
		}
		seen[path] = true
		if path, err = validateLLDB(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("llgo lldb: upstream LLDB %d or newer, or Apple LLDB not found; install LLDB or set LLGO_LLDB", minimumUpstreamLLDBVersion)
}

func validateLLDB(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("llgo lldb: find %q: %w", name, err)
	}
	output, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llgo lldb: query %q version: %w", path, err)
	}
	version, ok := parseLLDBVersion(string(output))
	if !ok {
		return "", fmt.Errorf("llgo lldb: cannot parse LLDB version from %q", strings.TrimSpace(string(output)))
	}
	// Apple reports an independent vendor build number (for example
	// lldb-1703), not the upstream LLVM major. Recognize that toolchain
	// explicitly instead of comparing its build number with the upstream
	// minimum.
	if !version.apple && version.major < minimumUpstreamLLDBVersion {
		return "", fmt.Errorf("llgo lldb: %q is upstream LLDB %d; version %d or newer is required", path, version.major, minimumUpstreamLLDBVersion)
	}
	return path, nil
}

func parseLLDBVersion(output string) (lldbVersion, bool) {
	pattern := upstreamLLDBVersionPattern
	apple := false
	match := pattern.FindStringSubmatch(output)
	if len(match) != 2 {
		pattern = appleLLDBVersionPattern
		apple = true
		match = pattern.FindStringSubmatch(output)
		if len(match) != 2 {
			return lldbVersion{}, false
		}
	}
	major, err := strconv.Atoi(match[1])
	return lldbVersion{major: major, apple: apple}, err == nil
}

func lldbImportCommand(path string) string {
	path = strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(path)
	return `command script import "` + path + `"`
}
