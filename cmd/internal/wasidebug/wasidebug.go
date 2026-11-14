// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

// Package wasidebug adapts LLDB's native Wasm JIT frames to the common Wasm
// runtime decoder. LLDB owns execution and all-stop shared-memory reads.
package wasidebug

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xgo-dev/llgo/cmd/internal/browser"
	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
)

//go:embed llgo_wasi.py
var adapter []byte

//go:embed decoder.js
var decoder []byte

// Prepare materializes a private session. Return paths are passed as argv, not
// interpolated into shell commands. The caller keeps directory alive in LLDB.
func Prepare(directory string, bundle *browserdebug.Bundle) (module, plugin string, err error) {
	if err = browser.WriteRuntimeDecoder(directory); err != nil {
		return
	}
	index, err := json.Marshal(bundle.Index)
	if err != nil {
		return "", "", err
	}
	raw, err := os.ReadFile(bundle.SymbolsPath)
	if err != nil {
		return "", "", err
	}
	for name, data := range map[string][]byte{
		"module.wasm":  raw,
		"index.json":   index,
		"schema.json":  debugabi.SchemaV1(),
		"llgo_wasi.py": adapter,
		"decoder.js":   decoder,
	} {
		if err = os.WriteFile(filepath.Join(directory, name), data, 0o600); err != nil {
			return "", "", err
		}
	}
	return filepath.Join(directory, "module.wasm"), filepath.Join(directory, "llgo_wasi.py"), nil
}
