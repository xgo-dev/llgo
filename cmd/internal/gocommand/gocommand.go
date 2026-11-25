/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package gocommand implements LLGo commands delegated to the Go toolchain.
package gocommand

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/gotool"
	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/env"
	"github.com/xgo-dev/llgo/cmd/compiler/mockable"
	"github.com/xgo-dev/llgo/target"
)

// Invocation describes one invocation of a command in the Go toolchain. Empty
// fields use the current process environment, executable, and standard streams.
type Invocation struct {
	Command    string
	Args       []string
	Env        []string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Executable string
}

// Run invokes the real Go tool while preventing a go-to-llgo alias from
// recursively starting LLGo again.
func (inv Invocation) Run() error {
	environ := inv.Env
	if environ == nil {
		environ = os.Environ()
	}
	self := inv.Executable
	if self == "" {
		var err error
		self, err = os.Executable()
		if err != nil {
			return err
		}
	}
	goExe, err := gotool.Find(self, envValue(environ, "PATH"))
	if err != nil {
		return err
	}
	cmd := exec.Command(goExe, append([]string{inv.Command}, inv.Args...)...)
	cmd.Stdin = defaultReader(inv.Stdin, os.Stdin)
	cmd.Stdout = defaultWriter(inv.Stdout, os.Stdout)
	cmd.Stderr = defaultWriter(inv.Stderr, os.Stderr)
	cmd.Env = gotool.ChildEnv(environ)
	return cmd.Run()
}

// Main runs a Go command without changing its arguments or environment.
func Main(command string, args []string) {
	Exit(command, (Invocation{Command: command, Args: args}).Run())
}

// BuildMain runs a Go command after applying LLGo source-selection rules.
func BuildMain(command string, args []string) {
	inv, err := Build(command, args)
	if err == nil {
		err = inv.Run()
	}
	Exit(command, err)
}

// Exit preserves an exit status already reported by Go. Errors produced before
// Go starts receive a stable LLGo command prefix.
func Exit(command string, err error) {
	if err == nil {
		return
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() > 0 {
		mockable.Exit(exit.ExitCode())
		return
	}
	fmt.Fprintf(os.Stderr, "llgo %s: %v\n", command, err)
	mockable.Exit(1)
}

// Build creates an invocation for a package-aware Go command. LLGo's -target
// flag is consumed here; target and user build tags are merged with LLGo's
// defaults, except for list module queries which do not select source files.
func Build(command string, args []string) (Invocation, error) {
	query, err := parseBuildArgs(args)
	if err != nil {
		return Invocation{}, err
	}
	environ := os.Environ()
	goos, goarch := envValue(environ, "GOOS"), envValue(environ, "GOARCH")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	var targetTags []string
	if query.target != "" {
		config, err := target.NewResolver(env.LLGoTargetsDir()).Resolve(query.target)
		if err != nil {
			return Invocation{}, err
		}
		if config.GOOS != "" {
			goos = config.GOOS
		}
		if config.GOARCH != "" {
			goarch = config.GOARCH
		}
		targetTags = config.BuildTags
	}
	var defaults []string
	if command != "list" || !query.module {
		defaults = splitTags(build.DefaultBuildTags())
	}
	if tags := mergeTags(defaults, targetTags, query.tags); len(tags) != 0 {
		query.goArgs, err = addBuildTags(query.goArgs, "-tags="+strings.Join(tags, ","))
		if err != nil {
			return Invocation{}, err
		}
	}
	return Invocation{
		Command: command,
		Args:    query.goArgs,
		Env:     replaceEnv(environ, "GOOS", goos, "GOARCH", goarch),
	}, nil
}

func addBuildTags(args []string, tags string) ([]string, error) {
	// Go requires -C to be the first flag. Keep it and its value ahead of the
	// injected tag flag instead of changing a previously valid invocation.
	insertAt := 0
	if len(args) != 0 {
		switch {
		case args[0] == "-C":
			if len(args) == 1 {
				return nil, errors.New("-C requires a value")
			}
			insertAt = 2
		case strings.HasPrefix(args[0], "-C="):
			insertAt = 1
		}
	}
	return slices.Insert(args, insertAt, tags), nil
}

type buildQuery struct {
	target string
	module bool
	tags   []string
	goArgs []string
}

func parseBuildArgs(args []string) (buildQuery, error) {
	var query buildQuery
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			query.goArgs = append(query.goArgs, args[index:]...)
			break
		}
		switch {
		case arg == "-target" || arg == "-tags":
			if index+1 == len(args) {
				return buildQuery{}, fmt.Errorf("%s requires a value", arg)
			}
			index++
			if arg == "-target" {
				query.target = args[index]
				if query.target == "" {
					return buildQuery{}, errors.New("-target requires a non-empty value")
				}
			} else {
				query.tags = append(query.tags, splitTags(args[index])...)
			}
		case strings.HasPrefix(arg, "-target="):
			query.target = strings.TrimPrefix(arg, "-target=")
			if query.target == "" {
				return buildQuery{}, errors.New("-target requires a non-empty value")
			}
		case strings.HasPrefix(arg, "-tags="):
			query.tags = append(query.tags, splitTags(strings.TrimPrefix(arg, "-tags="))...)
		case arg == "-m":
			query.module = true
			query.goArgs = append(query.goArgs, arg)
		case strings.HasPrefix(arg, "-m="):
			// Preserve Go's boolean spellings and leave invalid values for Go
			// to diagnose. Values consumed by -tags/-target never reach here.
			if enabled, err := strconv.ParseBool(strings.TrimPrefix(arg, "-m=")); err == nil {
				query.module = enabled
			}
			query.goArgs = append(query.goArgs, arg)
		default:
			query.goArgs = append(query.goArgs, arg)
		}
	}
	return query, nil
}

func splitTags(value string) []string {
	return strings.FieldsFunc(value, func(char rune) bool { return char == ',' || char == ' ' })
}

func mergeTags(groups ...[]string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, group := range groups {
		for _, tag := range group {
			if tag != "" && !seen[tag] {
				seen[tag] = true
				result = append(result, tag)
			}
		}
	}
	return result
}

func replaceEnv(environ []string, pairs ...string) []string {
	result := slices.Clone(environ)
	for index := 0; index < len(pairs); index += 2 {
		name, value := pairs[index], pairs[index+1]
		prefix := name + "="
		found := false
		for envIndex, entry := range result {
			if envNameEqual(entry, prefix) {
				result[envIndex] = prefix + value
				found = true
			}
		}
		if !found {
			result = append(result, prefix+value)
		}
	}
	return result
}

func envValue(environ []string, name string) string {
	prefix := name + "="
	for index := len(environ) - 1; index >= 0; index-- {
		if envNameEqual(environ[index], prefix) {
			return environ[index][len(prefix):]
		}
	}
	return ""
}

func envNameEqual(entry, prefix string) bool {
	if len(entry) < len(prefix) {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(entry[:len(prefix)], prefix)
	}
	return strings.HasPrefix(entry, prefix)
}

func defaultReader(value, fallback io.Reader) io.Reader {
	if value == nil {
		return fallback
	}
	return value
}

func defaultWriter(value, fallback io.Writer) io.Writer {
	if value == nil {
		return fallback
	}
	return value
}
