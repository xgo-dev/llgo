/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goplus/llgo/xtool/safesplit"
)

var (
	reSubcmd = regexp.MustCompile(`\$\([^)]+\)`)
	reFlag   = regexp.MustCompile(`[^ \t\n]+`)
)

func ExpandEnvToArgs(s string) []string {
	r, config := expandEnvWithCmd(s, "", nil)
	return expandedArgs(r, config)
}

// ExpandEnvToArgsWith expands variables and supported helper commands using
// the supplied request directory and environment. A non-nil environ prevents
// subprocesses and variable expansion from consulting process-global state.
func ExpandEnvToArgsWith(s, dir string, environ []string) []string {
	r, config := expandEnvWithCmd(s, dir, environ)
	return expandedArgs(r, config)
}

func expandedArgs(r string, config bool) []string {
	if r == "" {
		return nil
	}
	if config {
		return safesplit.SplitPkgConfigFlags(r)
	}
	return []string{r}
}

func ExpandEnv(s string) string {
	r, _ := expandEnvWithCmd(s, "", nil)
	return r
}

func expandEnvWithCmd(s, dir string, environ []string) (string, bool) {
	var config bool
	expanded := reSubcmd.ReplaceAllStringFunc(s, func(m string) string {
		subcmd := strings.TrimSpace(m[2 : len(m)-1])
		args := parseSubcmd(subcmd)
		cmd := args[0]
		if cmd != "pkg-config" && cmd != "llvm-config" {
			fmt.Fprintf(os.Stderr, "expand cmd only support pkg-config and llvm-config: '%s'\n", subcmd)
			return ""
		}
		config = true

		var out []byte
		var err error
		executable := cmd
		if environ != nil {
			executable = lookPathInEnvironment(cmd, dir, environ)
		}
		command := exec.Command(executable, args[1:]...)
		command.Dir = dir
		if environ != nil {
			command.Env = append([]string(nil), environ...)
		}
		out, err = command.Output()

		if err != nil {
			// TODO(kindy): log in verbose mode
			return ""
		}

		return strings.Replace(strings.TrimSpace(string(out)), "\n", " ", -1)
	})
	lookup := os.Getenv
	if environ != nil {
		lookup = func(key string) string {
			prefix := key + "="
			for i := len(environ) - 1; i >= 0; i-- {
				if strings.HasPrefix(environ[i], prefix) {
					return strings.TrimPrefix(environ[i], prefix)
				}
			}
			return ""
		}
	}
	return strings.TrimSpace(os.Expand(expanded, lookup)), config
}

func lookPathInEnvironment(name, dir string, environ []string) string {
	if strings.ContainsRune(name, filepath.Separator) {
		return name
	}
	path := ""
	prefix := "PATH="
	for i := len(environ) - 1; i >= 0; i-- {
		if strings.HasPrefix(environ[i], prefix) {
			path = strings.TrimPrefix(environ[i], prefix)
			break
		}
	}
	for _, entry := range filepath.SplitList(path) {
		if entry == "" {
			entry = "."
		}
		if !filepath.IsAbs(entry) && dir != "" {
			entry = filepath.Join(dir, entry)
		}
		candidate := filepath.Join(entry, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return name
}

func parseSubcmd(s string) []string {
	return reFlag.FindAllString(s, -1)
}
