/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package flags

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/compat"
	"github.com/xgo-dev/llgo/cmd/internal/gocommand"
	"github.com/xgo-dev/llgo/internal/quoted"
)

// ValidateMemProfileRate accepts the full-allocation mode implemented by LLGo.
// Size-class counters do not implement Go's probabilistic sampling intervals.
func ValidateMemProfileRate(value string) error {
	n, err := strconv.ParseInt(value, 0, 0)
	if err != nil {
		return fmt.Errorf("invalid memory profile rate: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("-memprofilerate=%s is unsupported: LLGo supports only 1 (full allocation counting)", value)
	}
	return nil
}

func rejectValue(fs *flag.FlagSet, name, reason string) {
	fs.Func(name, "Unsupported: "+reason, func(string) error {
		return fmt.Errorf("-%s is recognized but unsupported: %s", name, reason)
	})
}

func rejectBool(fs *flag.FlagSet, name, reason string) {
	fs.BoolFunc(name, "Unsupported: "+reason, func(string) error {
		return fmt.Errorf("-%s is recognized but unsupported: %s", name, reason)
	})
}

// AddExecFlag uses the Go command's quoted-word syntax. Wrappers are invoked
// directly with the linked artifact followed by program arguments, without a
// shell or target-template substitution.
func AddExecFlag(fs *flag.FlagSet) *[]string {
	words := new([]string)
	fs.Func("exec", "Execute the linked program using a quoted command and optional arguments", func(value string) error {
		parsed, err := quoted.Split(value)
		if err != nil {
			return err
		}
		*words = parsed
		return nil
	})
	return words
}

var buildFlagDescriptions = map[string]string{
	"p":             "Maximum number of concurrent LLGo build jobs",
	"gcflags":       "LLGo frontend options: -lang, -N, -l and global converthash settings (unpatterned or all=)",
	"ldflags":       "LLGo linker options: -s, -w, -compressdwarf, -extld, -extldflags (unpatterned or all=)",
	"mod":           "Module selection mode: readonly, vendor or mod",
	"modfile":       "Use an alternate module file for package selection",
	"modcacherw":    "Leave newly downloaded module cache directories writable",
	"overlay":       "Read Go source replacements from an overlay JSON file (non-Go replacements are unsupported)",
	"n":             "LLGo does not implement a build dry run",
	"work":          "LLGo does not implement retained work directories",
	"trimpath":      "LLGo does not implement artifact path trimming",
	"buildvcs":      "LLGo does not implement VCS stamping",
	"race":          "LLGo race instrumentation is not implemented",
	"msan":          "LLGo memory sanitizer instrumentation is not implemented",
	"asan":          "LLGo address sanitizer instrumentation is not implemented",
	"linkshared":    "LLGo cannot link gc shared Go packages",
	"compiler":      "LLGo uses the LLVM backend and cannot select a gc or gccgo compiler",
	"asmflags":      "LLGo does not forward options to its assembly backend",
	"gccgoflags":    "LLGo does not use the gccgo backend",
	"installsuffix": "LLGo does not use gc package installation directories",
	"pkgdir":        "LLGo cannot import gc object archives",
	"toolexec":      "LLGo does not wrap native compiler and linker invocations",
	"pgo":           "LLGo profile-guided optimization is not implemented",
}

func buildFlagHelp(name string) string { return buildFlagDescriptions[name] }

// FinalizeNativeFlags recognizes the remaining public Go controls as explicit
// rejections. This also catches them after package operands in llgo test,
// before an unknown flag can accidentally become a package-loader argument.
func FinalizeNativeFlags(command string, fs *flag.FlagSet) {
	for _, name := range compat.GoSurface().Commands[command] {
		record := compat.Flag(command, name)
		if fs.Lookup(name) != nil || record.Status != "rejected" {
			continue
		}
		reason := record.Note
		if specific := buildFlagHelp(name); specific != "" {
			reason = specific
		}
		switch name {
		case "a", "asan", "msan", "race", "linkshared", "n", "work", "trimpath", "json", "cover", "r", "i", "cache", "testcache", "modcache", "fuzzcache", "artifacts", "v", "m", "x":
			rejectBool(fs, name, reason)
		default:
			rejectValue(fs, name, reason)
		}
	}
}

// AddDirectoryFlag documents -C. ChangeDirectory consumes it before parsing,
// because all relative paths (including GOFLAGS paths) use the new directory.
func AddDirectoryFlag(fs *flag.FlagSet) {
	fs.Func("C", "Change directory before running the command; must be the first flag", func(string) error {
		return fmt.Errorf("-C must be the first flag")
	})
}

// ChangeDirectory applies only a leading -C; later occurrences are diagnosed
// by the flag parser instead of changing the meaning of earlier options.
func ChangeDirectory(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	name, dir, hasValue := strings.Cut(strings.TrimPrefix(strings.TrimPrefix(args[0], "-"), "-"), "=")
	if !strings.HasPrefix(args[0], "-") || name != "C" {
		return args, nil
	}
	consumed := 1
	if !hasValue {
		if len(args) == 1 {
			return nil, fmt.Errorf("-C requires a directory")
		}
		dir = args[1]
		consumed++
	}
	if dir == "" {
		return nil, fmt.Errorf("-C requires a non-empty directory")
	}
	if err := os.Chdir(dir); err != nil {
		return nil, fmt.Errorf("-C: %w", err)
	}
	return args[consumed:], nil
}

// Parse applies Go's environment defaults before explicit command flags. A
// separate parse keeps a GOFLAGS value from consuming a command-line operand.
// In particular, unsupported backend controls cannot bypass validation through
// GOFLAGS or a persistent GOENV file.
func Parse(fs *flag.FlagSet, args []string) error {
	args, err := ChangeDirectory(args)
	if err != nil {
		fmt.Fprintln(fs.Output(), err)
		return err
	}
	return ParseAfterDirectory(fs, args)
}

func ParseAfterDirectory(fs *flag.FlagSet, args []string) error {
	envArgs, err := defaults(fs)
	if err != nil {
		fmt.Fprintln(fs.Output(), err)
		return err
	}
	if err := fs.Parse(envArgs); err != nil {
		return err
	}
	return fs.Parse(args)
}

func defaults(fs *flag.FlagSet) ([]string, error) {
	value := os.Getenv("GOFLAGS")
	if value == "" && os.Getenv("GOENV") != "off" {
		var stdout, stderr bytes.Buffer
		if err := (gocommand.Invocation{Command: "env", Args: []string{"GOFLAGS"}, Stdout: &stdout, Stderr: &stderr}).Run(); err != nil {
			return nil, fmt.Errorf("read GOFLAGS: %s%w", stderr.String(), err)
		}
		value = strings.TrimSpace(stdout.String())
	}
	envArgs, err := quoted.Split(value)
	if err != nil {
		return nil, fmt.Errorf("parse GOFLAGS: %w", err)
	}
	var selected []string
	surface := compat.GoSurface()
	for _, arg := range envArgs {
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" || strings.HasPrefix(arg, "---") || strings.HasPrefix(arg, "-=") || strings.HasPrefix(arg, "--=") {
			return nil, fmt.Errorf("GOFLAGS contains non-flag %q", arg)
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name == "C" {
			return nil, fmt.Errorf("-C is not allowed in GOFLAGS")
		}
		f := fs.Lookup(name)
		if f == nil {
			// Go validates flag names across commands, then applies only the
			// defaults relevant to this command (for example, -run is ignored
			// by build). Native rejections remain registered and fail below.
			known := compat.Extensions[name] != "" || (strings.HasPrefix(name, "test.") &&
				slices.Contains(strings.Fields(compat.TestBinaryFlags), strings.TrimPrefix(name, "test.")))
			for _, names := range surface.Commands {
				known = known || slices.Contains(names, name)
			}
			if !known {
				return nil, fmt.Errorf("GOFLAGS contains unknown flag -%s", name)
			}
			continue
		}
		if !hasValue {
			isBool, ok := f.Value.(interface{ IsBoolFlag() bool })
			if !ok || !isBool.IsBoolFlag() {
				return nil, fmt.Errorf("GOFLAGS: -%s requires =value", name)
			}
		}
		selected = append(selected, arg)
	}
	return selected, nil
}
