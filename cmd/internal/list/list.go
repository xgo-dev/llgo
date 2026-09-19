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

// Package list implements the "llgo list" command.
package list

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

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/gotool"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/internal/targets"
)

var Cmd = &base.Command{
	UsageLine: "llgo list [-target name] [list flags] [packages]",
	Short:     "List packages using LLGo source-selection rules",
	Run:       runCmd,
}

func runCmd(_ *base.Command, args []string) {
	if err := run(args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() > 0 {
			mockable.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "llgo list:", err)
		mockable.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	query, err := parseArgs(args)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	goExe, err := gotool.Find(self, os.Getenv("PATH"))
	if err != nil {
		return err
	}

	goos, goarch := os.Getenv("GOOS"), os.Getenv("GOARCH")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	var targetTags []string
	if query.target != "" {
		config, err := targets.NewDefaultResolver().Resolve(query.target)
		if err != nil {
			return err
		}
		if config.GOOS != "" {
			goos = config.GOOS
		}
		if config.GOARCH != "" {
			goarch = config.GOARCH
		}
		targetTags = config.BuildTags
	}
	if tags := effectiveTags(query, goarch, targetTags); len(tags) != 0 {
		query.goArgs = append([]string{"-tags=" + strings.Join(tags, ",")}, query.goArgs...)
	}

	cmd := exec.Command(goExe, append([]string{"list"}, query.goArgs...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	cmd.Env = gotool.ChildEnv(replaceEnv(os.Environ(), "GOOS", goos, "GOARCH", goarch))
	return cmd.Run()
}

func effectiveTags(query listQuery, goarch string, targetTags []string) []string {
	if query.moduleMode {
		// Module queries omit LLGo defaults but retain target and user tags that
		// the caller explicitly requested.
		return mergeTags(targetTags, query.tags)
	}
	return mergeTags(strings.Split(build.DefaultBuildTags(goarch, query.target), ","), targetTags, query.tags)
}

type listQuery struct {
	target     string
	targetSet  bool
	tags       []string
	moduleMode bool
	goArgs     []string
}

func parseArgs(args []string) (listQuery, error) {
	var query listQuery
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			query.goArgs = append(query.goArgs, args[index:]...)
			break
		}
		switch {
		case arg == "-target" || arg == "-tags":
			if index+1 == len(args) {
				return listQuery{}, fmt.Errorf("%s requires a value", arg)
			}
			index++
			if arg == "-target" {
				query.target = args[index]
				query.targetSet = true
			} else {
				query.tags = append(query.tags, splitTags(args[index])...)
			}
		case strings.HasPrefix(arg, "-target="):
			query.target = strings.TrimPrefix(arg, "-target=")
			query.targetSet = true
		case strings.HasPrefix(arg, "-tags="):
			query.tags = append(query.tags, splitTags(strings.TrimPrefix(arg, "-tags="))...)
		default:
			if arg == "-m" {
				query.moduleMode = true
			} else if value, ok := strings.CutPrefix(arg, "-m="); ok {
				// Match the boolean spellings accepted by Go flags. Invalid values
				// remain forwarded so the real go command emits its diagnostic.
				if enabled, err := strconv.ParseBool(value); err == nil {
					query.moduleMode = enabled
				}
			}
			query.goArgs = append(query.goArgs, arg)
		}
	}
	if query.targetSet && query.target == "" {
		return listQuery{}, errors.New("-target requires a non-empty value")
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
			match := strings.HasPrefix(entry, prefix)
			if runtime.GOOS == "windows" {
				match = len(entry) >= len(prefix) && strings.EqualFold(entry[:len(prefix)], prefix)
			}
			if match {
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
