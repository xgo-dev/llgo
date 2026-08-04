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

package goflags

import (
	"strconv"
	"strings"

	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/optlevel"
)

// applyFrontendGCFlags maps the supported Go compiler flag subset to typed
// frontend configuration. Raw flags remain in GoBuildFlags for go/packages.
func applyFrontendGCFlags(conf *build.Config) {
	type frontendFlags struct {
		goVersion               string
		nNoOpt                  bool
		lNoOpt                  bool
		saturatingFloatToUint32 bool
		debug                   compilerDebugFlags
	}
	var applicable *frontendFlags
	for _, buildFlag := range conf.GoBuildFlags {
		value, ok := strings.CutPrefix(buildFlag, "-gcflags=")
		if !ok {
			continue
		}
		pattern, fields, err := splitPerPackageArgumentList(value)
		if err != nil || pattern != "" && pattern != "all" {
			// Raw package-specific values remain available to go/packages. LLGo
			// cannot map them to one global frontend configuration safely.
			continue
		}
		current := &frontendFlags{debug: make(compilerDebugFlags)}
		for _, compilerFlag := range fields {
			switch {
			case strings.HasPrefix(compilerFlag, "-lang="):
				current.goVersion = strings.TrimPrefix(compilerFlag, "-lang=")
			case compilerFlag == "-N":
				current.nNoOpt = true
			case strings.HasPrefix(compilerFlag, "-N="):
				current.nNoOpt = countFlagEnabled(strings.TrimPrefix(compilerFlag, "-N="))
			case compilerFlag == "-l":
				current.lNoOpt = true
			case strings.HasPrefix(compilerFlag, "-l="):
				current.lNoOpt = countFlagIsOne(strings.TrimPrefix(compilerFlag, "-l="))
			case strings.HasPrefix(compilerFlag, "-d="):
				current.debug.apply(strings.TrimPrefix(compilerFlag, "-d="))
			}
		}
		current.saturatingFloatToUint32 = bisectPatternAlwaysEnabled(current.debug["converthash"])
		applicable = current
	}
	if applicable == nil {
		return
	}
	if applicable.goVersion != "" {
		conf.GoVersion = applicable.goVersion
	}
	if applicable.nNoOpt || applicable.lNoOpt {
		conf.OptLevel = optlevel.O0
	}
	conf.SaturatingFloatToUint32 = applicable.saturatingFloatToUint32
}

// compilerDebugFlags stores the last value of each comma-separated -d setting.
// The syntax is shared by all cmd/compile debug settings; a setting without an
// explicit value is equivalent to setting it to 1.
type compilerDebugFlags map[string]string

func (f compilerDebugFlags) apply(list string) {
	for _, setting := range strings.Split(list, ",") {
		name, value, ok := strings.Cut(setting, "=")
		if !ok {
			value = "1"
		}
		if name != "" {
			f[name] = value
		}
	}
}

// bisectPatternAlwaysEnabled reports whether a Go internal/bisect pattern is
// one of the global forms that enables every candidate. Hash-debug settings
// such as converthash, fmahash, and loopvarhash all use this syntax. LLGo can
// map these global forms to a build-wide option; selective hash patterns remain
// forwarded to go/packages but cannot be represented by that option.
func bisectPatternAlwaysEnabled(pattern string) bool {
	// q suppresses match reports but does not change which candidates match.
	if strings.HasPrefix(pattern, "q") {
		pattern = pattern[1:]
	}
	// v requests visible reports and may be repeated by the bisect driver.
	for strings.HasPrefix(pattern, "v") {
		pattern = pattern[1:]
	}

	enable := true
	for strings.HasPrefix(pattern, "!") {
		enable = !enable
		pattern = pattern[1:]
	}
	if pattern == "n" { // n is the bisect alias for !y.
		enable = !enable
		pattern = "y"
	}
	return enable && pattern == "y"
}

func countFlagEnabled(value string) bool {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	v, err := strconv.Atoi(value)
	return err == nil && v != 0
}

func countFlagIsOne(value string) bool {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	v, err := strconv.Atoi(value)
	return err == nil && v == 1
}
