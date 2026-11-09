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
	"flag"
	"fmt"
	"go/version"
	"io"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

type frontendGCFlags struct {
	goVersion        string
	nCount           int
	lCount           int
	mCount           int
	dependencyMCount int
	debug            compilerDebugFlags
}

// parseFrontendGCFlags validates and captures the supported compiler options
// in one pass, including space-separated values and double-dash spellings.
func parseFrontendGCFlags(fields []string) (*frontendGCFlags, error) {
	parsed := &frontendGCFlags{debug: make(compilerDebugFlags)}
	fs := flag.NewFlagSet("gcflags", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Func("lang", "Go language version", func(v string) error {
		if !version.IsValid(v) {
			return fmt.Errorf("invalid Go language version %q", v)
		}
		parsed.goVersion = v
		return nil
	})
	fs.BoolFunc("N", "Disable optimization", func(v string) error {
		return applyCompilerCount(v, &parsed.nCount)
	})
	fs.BoolFunc("l", "Disable inlining", func(v string) error {
		return applyCompilerCount(v, &parsed.lCount)
	})
	fs.BoolFunc("m", "Print allocation decisions; repeat for reasons", func(v string) error {
		return applyCompilerCount(v, &parsed.mCount)
	})
	fs.Func("d", "Global float-to-integer conversion behavior", func(v string) error {
		for _, setting := range strings.Split(v, ",") {
			name, pattern, ok := strings.Cut(setting, "=")
			if name != "converthash" || !ok || !globalBisectPattern(pattern) {
				return fmt.Errorf("gc debug setting %q is unsupported", setting)
			}
		}
		parsed.debug.apply(v)
		return nil
	})
	if err := fs.Parse(fields); err != nil {
		return nil, fmt.Errorf("unsupported LLGo compiler option: %w", err)
	}
	if fs.NArg() != 0 {
		return nil, fmt.Errorf("unexpected compiler operands %q", fs.Args())
	}
	if parsed.lCount != 0 && parsed.lCount != 1 {
		return nil, fmt.Errorf("gc inliner diagnostic levels are unsupported")
	}
	return parsed, nil
}

// apply maps exactly the compiler options accepted by the parser to typed
// frontend configuration. Raw flags remain in GoBuildFlags for go/packages.
func (f *frontendGCFlags) apply(conf *build.Config) {
	if f == nil {
		return
	}
	if f.goVersion != "" {
		conf.GoVersion = f.goVersion
	}
	if f.nCount != 0 || f.lCount == 1 {
		conf.OptLevel = optlevel.O0
	}
	conf.SaturatingFloatToInt = bisectPatternAlwaysEnabled(f.debug["converthash"])
	conf.EscapeDiagnostics = f.mCount
	conf.DependencyEscapeDiagnostics = f.dependencyMCount
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

func globalBisectPattern(pattern string) bool {
	pattern = strings.TrimPrefix(pattern, "q")
	pattern = strings.TrimLeft(pattern, "v")
	pattern = strings.TrimLeft(pattern, "!")
	return pattern == "y" || pattern == "n"
}

// applyCompilerCount follows cmd/compile's count flags: a bare flag or true
// increments the count, false resets it, and an integer replaces it.
func applyCompilerCount(value string, count *int) error {
	switch value {
	case "true":
		*count++
	case "false":
		*count = 0
	default:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid count %q", value)
		}
		*count = n
	}
	return nil
}
