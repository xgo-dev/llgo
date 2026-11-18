// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package debug

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/xgo-dev/llgo/cmd/internal/lldb"
	"github.com/xgo-dev/llgo/cmd/internal/wasidebug"
	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

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

func wasiLLDBArguments(host, module string, extra []string) []string {
	args := []string{"-O", "settings set plugin.jit-loader.gdb.enable on"}
	args = append(args, extra...)
	return append(args, "--", host, module)
}

func runWASISession(s session, stdin io.Reader, stdout, stderr io.Writer) error {
	runtime, err := findWASIHost(s.options.wasiHost)
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
