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

// Package compat records LLGo's public command contract. The Go snapshot is
// reviewed whenever the pinned toolchain changes; Go flags and LLGo extensions
// have separate records so delegation never implies native backend semantics.
package compat

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed go1.27.json
var data embed.FS

type Record struct {
	Status string
	Owner  string
	Note   string
}

type Surface struct {
	GoVersion      string              `json:"go_version"`
	TestFlagPrefix string              `json:"test_flag_prefix"`
	Commands       map[string][]string `json:"commands"`
}

func GoSurface() Surface {
	bytes, err := data.ReadFile("go1.27.json")
	if err != nil {
		panic(err)
	}
	var surface Surface
	if err := json.Unmarshal(bytes, &surface); err != nil {
		panic(err)
	}
	return surface
}

func Command(name string) Record {
	if name == "tool compile" {
		return Record{"native", "cmd/internal/compile", "LLGo source validation and IR generation; gc object and backend controls are rejected"}
	}
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return Record{}
	}
	switch fields[0] {
	case "fmt", "doc", "mod", "work":
		return Record{"delegated", "cmd/internal/gocommand", "Exact Go delegation; no LLGo artifact or runtime semantics"}
	case "fix", "generate", "get", "list", "vet":
		return Record{"delegated", "cmd/internal/gocommand", "Go delegation with LLGo source tags and target selection"}
	case "env":
		return Record{"native", "cmd/internal/env", "Go environment delegation plus LLGo-specific queries"}
	case "telemetry":
		return Record{"omitted", "cmd/llgo", "Go telemetry describes the Go toolchain rather than LLGo"}
	case "bug":
		return Record{"native", "cmd/internal/bug", "LLGo issue tracker; allowlisted version/platform diagnostics"}
	case "version":
		return Record{"native", "cmd/internal/version", "LLGo identity; file inspection delegates to Go and requires Go build information"}
	case "tool":
		return Record{"native", "cmd/llgo/tool_cmd.gox", "LLGo tools only; gc object/ABI tools are not delegated"}
	case "cmptest":
		return Record{"native", "cmd/internal/run", "LLGo comparison workflow"}
	case "build", "install", "clean", "run", "test":
		return Record{"native", "cmd/internal/" + name, "Native LLGo workflow; only explicitly classified flags have implemented semantics"}
	default:
		return Record{}
	}
}

// Flag returns an explicit record for every public flag of a shared command.
// A rejected record may be an explicit unsupported diagnostic or the standard
// undefined-flag error; neither may successfully parse and discard a value.
func Flag(command, name string) Record {
	if command == "test" && strings.HasPrefix(name, "test.") {
		short := strings.TrimPrefix(name, "test.")
		if !in(short, TestBinaryFlags) {
			return Record{"rejected", "cmd/internal/test", "Only Go test-binary flags have a test. alias"}
		}
		return Flag(command, short)
	}
	c := Command(command)
	if c.Status == "delegated" || command == "env" {
		return Record{"delegated", c.Owner, c.Note}
	}
	rejected := Record{"rejected", c.Owner, "Not implemented by this native LLGo command"}
	if command == "tool compile" {
		if in(name, "B C N V c complete l lang") {
			return Record{"native", c.Owner, "LLGo frontend, optimization, concurrency or identity; -l accepts only 0/1"}
		}
		if in(name, "e std") {
			return Record{"native", "internal/packages", "Source-loader diagnostics or standard-library pragma validation; no gc backend effect"}
		}
		if name == "d" {
			return Record{"partial", c.Owner, "panic and SSA-check harness settings only; LLGo SSA checks are always enabled; gc settings are rejected"}
		}
		return Record{"rejected", c.Owner, "gc object format, backend or diagnostics are not implemented by LLGo"}
	}
	if command == "version" {
		if name == "C" {
			return Record{"native", "cmd/internal/flags", "Changes directory before parsing relative paths"}
		}
		if in(name, "m v json") {
			return Record{"delegated", "cmd/internal/gocommand", "File inspection, -m and -json delegate to Go; -v alone retains LLGo identity"}
		}
		return rejected
	}
	if command == "clean" {
		if in(name, "C tags v") {
			return Record{"native", "cmd/internal/clean", "Package selection and cleanup output"}
		}
		return rejected
	}
	if !in(command, "build install run test cmptest") {
		return rejected
	}
	if in(name, "C a p v x tags mod modfile modcacherw") {
		return Record{"native", "cmd/internal/flags", "Typed native build configuration or source graph selection"}
	}
	if name == "exec" && in(command, "run test") {
		return Record{"native", "internal/build", "Quoted external execution wrapper, overriding the default target runner"}
	}
	if name == "overlay" {
		return Record{"partial", "internal/goflags", "Go source replacement, addition and deletion; non-Go replacements are explicitly rejected"}
	}
	if in(name, "gcflags ldflags") {
		return Record{"partial", "internal/goflags", "Only the documented portable subset; other values and per-package patterns are rejected"}
	}
	if name == "buildmode" && in(command, "build test") {
		return Record{"partial", "cmd/internal/flags", "Only exe, c-archive and c-shared; test execution requires exe, library test artifacts require -c; gc ABI modes are rejected"}
	}
	if command == "build" && in(name, "o cover covermode coverpkg") {
		return Record{"native", "cmd/internal/build", "Artifact output, coverage or C ABI build modes"}
	}
	if command == "test" {
		if in(name, "cpuprofile memprofile") {
			return Record{"partial", "cmd/internal/test", "Qualified native runtimes only; CPU sampling requires a supported sampler; retains one test binary; allocation profiles count size classes"}
		}
		if in(name, "c o cover covermode coverpkg coverprofile args artifacts bench benchmem benchtime count failfast fullpath json list outputdir parallel run short shuffle skip timeout") {
			return Record{"native", "cmd/internal/test", "Test driver, LLGo testing runtime or pprof output"}
		}
		if name == "cpu" {
			return Record{"rejected", "runtime/internal/lib/runtime", "Native GOMAXPROCS execution limiting is not implemented"}
		}
		if in(name, "fuzz fuzztime fuzzminimizetime") {
			return Record{"rejected", "cmd/internal/test", "LLGo fuzz instrumentation, minimization, crash and cache behavior are not qualified"}
		}
		if name == "memprofilerate" {
			return Record{"partial", "runtime/internal/runtime", "Only 1 (full size-class allocation counting); other sampling intervals are rejected"}
		}
	}
	return rejected
}

// TestBinaryFlags is the public Go 1.27 set accepting a test. prefix. Driver
// flags such as -json and -cover are deliberately excluded.
const TestBinaryFlags = "artifacts bench benchmem benchtime blockprofile blockprofilerate count coverprofile cpu cpuprofile failfast fullpath fuzz fuzzminimizetime fuzztime list memprofile memprofilerate mutexprofile mutexprofilefraction outputdir parallel run short shuffle skip timeout trace v"

// Extensions lists names outside the Go contract, with their actual consumers.
// Command-specific registration is checked separately against these names.
var Extensions = map[string]string{
	"n": "cmd/internal/bug", "v": "internal/monitor", "C": "cmd/internal/flags", "O": "cmd/internal/flags", "O0": "cmd/internal/flags", "O1": "cmd/internal/flags", "O2": "cmd/internal/flags", "O3": "cmd/internal/flags", "Os": "cmd/internal/flags", "Oz": "cmd/internal/flags",
	"compiler-verbose": "cmd/internal/flags", "cv": "cmd/internal/flags", "lto": "cmd/internal/flags", "lto-pass-plugin": "cmd/internal/flags", "globaldce": "cmd/internal/flags",
	"pclntab": "cmd/internal/flags", "debug-artifact": "cmd/internal/flags", "deadcodedrop": "cmd/internal/flags", "goroutine-stack-size": "cmd/internal/flags", "pthread-stack-size": "cmd/internal/flags",
	"check-linkargs": "internal/build", "check-llfiles": "internal/build", "gen-llfiles": "internal/build", "force-espclang": "internal/build",
	"debug-trace": "internal/build", "target": "internal/crosscompile", "emulator": "internal/build", "port": "internal/flash", "baudrate": "internal/monitor",
	"size": "internal/build", "size-format": "internal/build", "size-level": "internal/build", "obin": "internal/build", "ohex": "internal/build", "oimg": "internal/build", "ouf2": "internal/build", "ozip": "internal/build",
	"timeout": "internal/build", "gen": "internal/build",
	"paniconexit0": "cmd/internal/test", "testlogfile": "cmd/internal/test", "gocoverdir": "cmd/internal/test", "fuzzworker": "cmd/internal/test", "fuzzcachedir": "cmd/internal/test",
}

func in(name, list string) bool {
	for _, item := range strings.Fields(list) {
		if item == name {
			return true
		}
	}
	return false
}
