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

// Package debug implements the cross-platform "llgo debug" command.
package debug

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/cmd/internal/browser"
	"github.com/xgo-dev/llgo/cmd/internal/flags"
	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/mockable"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/internal/targets"
)

// Cmd is the llgo debug command.
var Cmd = &base.Command{
	UsageLine: "llgo debug [-backend auto|lldb|gdb|browser] [-target platform] [build flags] [package] [-- debugger arguments...]",
	Short:     "Build and debug an LLGo program",
}

var (
	goBuildFlags    *base.PassArgs
	backendFlag     string
	lldbPath        string
	gdbPath         string
	remoteAddress   string
	serverCommand   string
	chromePath      string
	browserDevtools bool
	sourceMaps      sourceMapsFlag
	loadImage       bool
)

func init() {
	Cmd.Run = runCmd
	goBuildFlags = flags.CaptureGoBuildFlags(Cmd)
	flags.AddCommonFlags(&Cmd.Flag)
	flags.AddCompilerVerboseFlag(&Cmd.Flag)
	flags.AddBuildFlags(&Cmd.Flag)
	flags.AddEmbeddedFlags(&Cmd.Flag)
	flags.AddOutputFlags(&Cmd.Flag)
	Cmd.Flag.StringVar(&backendFlag, "backend", string(backendAuto), "debug backend: auto, lldb, gdb, or browser; wasmtime is reserved and unavailable")
	Cmd.Flag.StringVar(&lldbPath, "lldb", "", "path to LLDB (default $LLGO_LLDB or auto-detect)")
	Cmd.Flag.StringVar(&gdbPath, "gdb", "", "path to GDB (default $LLGO_GDB, target candidates, or auto-detect)")
	Cmd.Flag.StringVar(&remoteAddress, "remote", "", "connect to an existing debug server at host:port")
	Cmd.Flag.StringVar(&serverCommand, "server", "", "debug-server template: {} is the artifact, {debug-stdio} selects RSP over stdin/stdout, {debug-port} selects legacy TCP")
	Cmd.Flag.BoolVar(&loadImage, "load", false, "reset, load the image, and halt an existing remote or custom debug server")
	Cmd.Flag.StringVar(&chromePath, "chrome", "", "path to Chrome for Testing or Chromium (default $LLGO_CHROME or auto-detect)")
	Cmd.Flag.BoolVar(&browserDevtools, "browser-devtools", true, "open browser DevTools automatically")
	Cmd.Flag.Var(&sourceMaps, "source-map", "map a recorded source prefix to local files: FROM=TO (repeatable)")
}

func runCmd(cmd *base.Command, args []string) {
	commandArgs, debuggerArgs := splitDebuggerArgs(args)
	if err := cmd.Flag.Parse(commandArgs); err != nil {
		mockable.Exit(2)
		return
	}
	if err := run(cmd.Flag.Args(), debuggerArgs, options{
		backend: backend(backendFlag),
		lldb:    lldbPath,
		gdb:     gdbPath,
		remote:  remoteAddress,
		server:  serverCommand,
		load:    loadImage,
		browser: browser.Options{Chrome: chromePath, DisableTools: !browserDevtools, SourceMaps: sourceMaps},
	}, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		mockable.Exit(1)
	}
}

type sourceMapsFlag []browserdebug.PathMapping

func (m *sourceMapsFlag) String() string {
	var values []string
	for _, mapping := range *m {
		values = append(values, mapping.From+"="+mapping.To)
	}
	return strings.Join(values, ", ")
}

func (m *sourceMapsFlag) Set(value string) error {
	mapping, err := browserdebug.ParsePathMapping(value)
	if err != nil {
		return err
	}
	*m = append(*m, mapping)
	return nil
}

func splitDebuggerArgs(args []string) (command, debugger []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func run(packageArgs, debuggerArgs []string, opts options, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(packageArgs) > 1 {
		return errors.New("llgo debug: exactly one package may be debugged")
	}
	if len(packageArgs) == 0 {
		packageArgs = []string{"."}
	}
	if err := opts.validate(); err != nil {
		return err
	}

	conf := build.NewDefaultConf(build.ModeBuild)
	if err := flags.UpdateConfig(conf); err != nil {
		return fmt.Errorf("llgo debug: %w", err)
	}
	if err := flags.ApplyGoBuildFlags(conf, goBuildFlags.Args); err != nil {
		return fmt.Errorf("llgo debug: %w", err)
	}
	conf.BuildMode = build.BuildModeExe
	conf.OmitDWARFByDefault = false
	if conf.LinkOptions.EffectiveOmitDWARF() ||
		(conf.DebugArtifactModeSet && conf.DebugArtifactMode == build.DebugArtifactNone) {
		return errors.New("llgo debug: debug information is required; remove -ldflags=-w or -debug-artifact=none")
	}
	// Debug sessions require DWARF even when the selected target omits it by
	// default. Keep explicit stripping requests above as user-facing errors.
	conf.LinkOptions.DWARF = build.DWARFPreserve
	if conf.OptLevel == optlevel.Unset {
		conf.OptLevel = optlevel.O0
	}

	target, err := resolveTarget(conf.Target)
	if err != nil {
		return err
	}
	if target != nil {
		if target.GOOS != "" {
			conf.Goos = target.GOOS
		}
		if target.GOARCH != "" {
			conf.Goarch = target.GOARCH
		}
	}
	selected, err := selectBackend(opts.backend, classifyTarget(conf, target))
	if err != nil {
		return err
	}
	if err := validateSessionTarget(conf, target, selected, opts); err != nil {
		return err
	}
	if selected == backendBrowser && !slices.Contains(strings.Fields(strings.ReplaceAll(conf.Tags, ",", " ")), "llgo.wasm.debugger") {
		if conf.Tags != "" {
			conf.Tags += ","
		}
		conf.Tags += "llgo.wasm.debugger"
	}

	cleanup, artifact, err := prepareArtifact(conf)
	if err != nil {
		return err
	}
	defer cleanup()
	built, err := build.Do(packageArgs, conf)
	if err != nil {
		return err
	}
	if selected == backendBrowser {
		artifact = browserModulePath(artifact)
		opts.browser.SourceRoots = append(opts.browser.SourceRoots, builtSourceRoots(built)...)
	}
	if _, err = os.Stat(artifact); err != nil {
		return fmt.Errorf("llgo debug: built artifact %q is unavailable: %w", artifact, err)
	}

	return runSession(session{
		backend:      selected,
		artifact:     artifact,
		debuggerArgs: debuggerArgs,
		target:       target,
		options:      opts,
	}, stdin, stdout, stderr)
}

// Source trust comes from packages selected by this build, never paths claimed
// by an artifact's DWARF. Standalone StartSession callers must supply mappings
// or explicit roots for sources outside the artifact directory.
func builtSourceRoots(built []build.Package) []string {
	seen := make(map[*packages.Package]bool)
	directories := make(map[string]bool)
	var visit func(*packages.Package)
	visit = func(pkg *packages.Package) {
		if pkg == nil || seen[pkg] {
			return
		}
		seen[pkg] = true
		for _, path := range append(append([]string(nil), pkg.GoFiles...), pkg.OtherFiles...) {
			directories[filepath.Dir(path)] = true
		}
		for _, dependency := range pkg.Imports {
			visit(dependency)
		}
	}
	for _, pkg := range built {
		if pkg != nil {
			visit(pkg.Package)
		}
	}
	var roots []string
	for directory := range directories {
		roots = append(roots, directory)
	}
	return roots
}

func validateSessionTarget(conf *build.Config, target *targets.Config, selected backend, opts options) error {
	if err := validateBackendOptions(selected, opts); err != nil {
		return err
	}
	if selected != backendBrowser && target == nil && opts.remote == "" && (conf.Goos != runtime.GOOS || conf.Goarch != runtime.GOARCH) {
		return fmt.Errorf("llgo debug: cannot launch a %s/%s program on %s/%s without -remote", conf.Goos, conf.Goarch, runtime.GOOS, runtime.GOARCH)
	}
	return nil
}

func browserModulePath(artifact string) string {
	switch ext := filepath.Ext(artifact); ext {
	case ".js", ".mjs", ".html":
		return strings.TrimSuffix(artifact, ext) + ".wasm"
	default:
		return artifact
	}
}

func resolveTarget(name string) (*targets.Config, error) {
	if name == "" {
		return nil, nil
	}
	target, err := targets.NewDefaultResolver().Resolve(name)
	if err != nil {
		return nil, fmt.Errorf("llgo debug: %w", err)
	}
	return target, nil
}

func prepareArtifact(conf *build.Config) (cleanup func(), artifact string, err error) {
	cleanup = func() {}
	ext := debugArtifactExtension(conf)
	if conf.Goos == "js" {
		switch requested := filepath.Ext(conf.OutFile); requested {
		case ".js", ".mjs", ".html":
			ext = requested
		}
	}
	if conf.OutFile == "" {
		dir, err := os.MkdirTemp("", "llgo-debug-")
		if err != nil {
			return cleanup, "", fmt.Errorf("llgo debug: create artifact directory: %w", err)
		}
		cleanup = func() { os.RemoveAll(dir) }
		conf.OutFile = filepath.Join(dir, "program"+ext)
	} else if ext != "" && !strings.HasSuffix(conf.OutFile, ext) {
		conf.OutFile += ext
	}
	conf.AppExt = ext
	artifact, err = filepath.Abs(conf.OutFile)
	if err != nil {
		cleanup()
		return func() {}, "", fmt.Errorf("llgo debug: resolve artifact path: %w", err)
	}
	conf.OutFile = artifact
	return cleanup, artifact, nil
}

func debugArtifactExtension(conf *build.Config) string {
	if conf.Target != "" {
		if conf.Goarch == "wasm" || strings.HasPrefix(conf.Target, "wasi") || strings.HasPrefix(conf.Target, "wasm") || strings.HasPrefix(conf.Target, "emscripten") {
			return ".wasm"
		}
		return ".elf"
	}
	switch conf.Goos {
	case "windows":
		return ".exe"
	case "js", "wasi", "wasip1":
		return ".wasm"
	default:
		return ""
	}
}
