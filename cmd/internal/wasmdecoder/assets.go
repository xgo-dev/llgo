// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

// Package wasmdecoder supplies shared Wasm runtime value and stack decoders to
// debugger transports. It has no browser-launch or execution-control dependency.
package wasmdecoder

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed plugin.js
var plugin []byte

//go:embed goroutines.js
var goroutines []byte

// Write materializes the decoder scripts in an existing session directory.
func Write(directory string, mode os.FileMode) error {
	for name, data := range map[string][]byte{"plugin.js": plugin, "goroutines.js": goroutines} {
		if err := os.WriteFile(filepath.Join(directory, name), data, mode); err != nil {
			return err
		}
	}
	return nil
}
