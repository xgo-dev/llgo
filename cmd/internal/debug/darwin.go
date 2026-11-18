// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package debug

import (
	"debug/macho"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Darwin's LTO debug map can name one source file for an object containing
// many compile units. LLDB can find functions while failing file/line
// breakpoints. Materialize a private dSYM for the session, leaving the linked
// executable and its platform-default debug packaging untouched.
func darwinLTOSymbols(artifact string, enabled bool) (arguments []string, cleanup func(), err error) {
	cleanup = func() {}
	if runtime.GOOS != "darwin" || !enabled {
		return
	}
	image, err := macho.Open(artifact)
	if err != nil {
		return nil, cleanup, fmt.Errorf("llgo debug: read Darwin LTO debug map: %w", err)
	}
	defer image.Close()
	hasMap := false
	if image.Symtab != nil {
		for _, symbol := range image.Symtab.Syms {
			if symbol.Type == 0x66 { // N_OSO: retained object in a Darwin debug map.
				hasMap = true
				break
			}
		}
	}
	// A stripped executable uses its adjacent dSYM. Historical LTO directories
	// are not evidence of this build's mode or of an available debug map.
	if !hasMap {
		return nil, cleanup, nil
	}
	directory, err := os.MkdirTemp("", "llgo-debug-dsym-")
	if err != nil {
		return nil, cleanup, err
	}
	cleanup = func() { os.RemoveAll(directory) }
	symbols := filepath.Join(directory, "program.dSYM")
	output, err := exec.Command("dsymutil", artifact, "-o", symbols).CombinedOutput()
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("llgo debug: prepare Darwin LTO symbols with dsymutil: %w\n%s", err, output)
	}
	file, err := macho.Open(filepath.Join(symbols, "Contents", "Resources", "DWARF", filepath.Base(artifact)))
	if err == nil {
		defer file.Close()
		if section := file.Section("__debug_info"); section == nil || section.Size == 0 {
			err = fmt.Errorf("empty dSYM; retained LTO debug objects are unavailable")
		}
	}
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("llgo debug: read Darwin LTO symbols: %w\n%s", err, output)
	}
	return []string{"-o", "target symbols add " + quoteLLDBArgument(symbols)}, cleanup, nil
}
