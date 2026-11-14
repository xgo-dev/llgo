// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package debug

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/xgo-dev/llgo/cmd/internal/lldb"
	"github.com/xgo-dev/llgo/cmd/internal/wasidebug"
	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
	"golang.org/x/mod/semver"
)

// 46 is the final Wasmtime branch implementing the WASI threads proposal.
// Keep this capability separate from Wasm atomics/threads, which newer engines
// still implement without the required wasi.thread-spawn host interface.
func threadedWasmtimeVersion(output string) (string, error) {
	fields := strings.Fields(output)
	if len(fields) < 2 || fields[0] != "wasmtime" {
		return "", fmt.Errorf("unexpected Wasmtime version output %q", strings.TrimSpace(output))
	}
	version := "v" + strings.TrimPrefix(fields[1], "v")
	if !semver.IsValid(version) || semver.Compare(version, "v46.0.3") < 0 || semver.Major(version) != "v46" {
		return "", fmt.Errorf("W32 threaded source debugging requires Wasmtime 46.0.3 or a later 46.x patch; found %s (47+ removed WASI threads); use -wasmtime or LLGO_WASMTIME", fields[1])
	}
	return version, nil
}

func findThreadedWasmtime(configured string) (string, error) {
	if configured == "" {
		configured = os.Getenv("LLGO_WASMTIME")
	}
	if configured == "" {
		configured = "wasmtime"
	}
	path, err := exec.LookPath(configured)
	if err != nil {
		return "", fmt.Errorf("llgo debug: find Wasmtime: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llgo debug: query Wasmtime version: %w: %s", err, output)
	}
	if _, err := threadedWasmtimeVersion(string(output)); err != nil {
		return "", fmt.Errorf("llgo debug: %w", err)
	}
	return path, nil
}

// Wasmtime reads DWARF from its input module. For external packaging, use the
// debugger-owned full module only after verifying that removing its DWARF
// reproduces the deployed module exactly, including code/data and metadata.
func wasiDebugModule(path string, mappings []browserdebug.PathMapping) (*browserdebug.Bundle, error) {
	bundle, err := browserdebug.Load(path, mappings)
	if err != nil {
		return nil, fmt.Errorf("llgo debug: read WASI debug artifact: %w", err)
	}
	if bundle.Index.Record.PointerSize != 4 || bundle.Index.Record.ByteOrder != debugabi.ByteOrderLittle {
		return nil, errors.New("llgo debug: WASI source debugging requires the little-endian W32 profile")
	}
	if bundle.MainPath != bundle.SymbolsPath {
		main, err := os.ReadFile(bundle.MainPath)
		if err != nil {
			return nil, err
		}
		symbols, err := os.ReadFile(bundle.SymbolsPath)
		if err != nil {
			return nil, err
		}
		reference, _, err := wasmdebug.ExternalURL(main)
		if err != nil {
			return nil, err
		}
		expected, err := wasmdebug.Externalize(symbols, reference)
		if err != nil || !bytes.Equal(expected, main) {
			return nil, errors.New("llgo debug: external WASI debug artifact does not match deployed code/data")
		}
	}
	return bundle, nil
}

func wasiLLDBArguments(runtime, module string, extra []string) []string {
	args := []string{"-O", "settings set plugin.jit-loader.gdb.enable on"}
	args = append(args, extra...)
	// JIT registration happens before Wasmtime serializes a cache miss. An
	// attached debugger can already have patched code with breakpoints; never
	// cache or reuse those instrumented native instructions in source sessions.
	return append(args, "--", runtime, "run", "-Ccache=n", "-Ddebug-info=y", "-Oopt-level=0",
		"-Sthreads=y", "-Wthreads=y,shared-memory=y,exceptions=y", module)
}

func runWASISession(s session, stdin io.Reader, stdout, stderr io.Writer) error {
	runtime, err := findThreadedWasmtime(s.options.wasmtime)
	if err != nil {
		return err
	}
	bundle, err := wasiDebugModule(s.artifact, s.options.browser.SourceMaps)
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "llgo-wasi-debug-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	module, plugin, err := wasidebug.Prepare(directory, bundle)
	if err != nil {
		return err
	}
	extra := []string{"-O", "command script import " + strconv.Quote(plugin)}
	extra = append(extra, s.debuggerArgs...)
	args := wasiLLDBArguments(runtime, module, extra)
	return lldb.Run(s.options.lldb, args, stdin, stdout, stderr)
}
