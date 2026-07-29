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

// Package wasmworkers validates the bounded WebAssembly worker-pool setting.
package wasmworkers

import (
	"fmt"
	"path/filepath"
	"strconv"
)

const (
	DefaultCount = 1
	MaxCount     = 16
)

type Config struct {
	Count int
}

func Parse(value string) (Config, error) {
	if value == "" {
		return Config{Count: DefaultCount}, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 1 || count > MaxCount {
		return Config{}, fmt.Errorf("LLGO_WASM_WORKERS must be an integer from 1 through %d", MaxCount)
	}
	return Config{Count: count}, nil
}

func (c Config) Enabled() bool {
	return c.Count > DefaultCount
}

func (c Config) ValidateTarget(goos, goarch string) error {
	if !c.Enabled() {
		return nil
	}
	if goos != "js" || goarch != "wasm" {
		return fmt.Errorf("LLGO_WASM_WORKERS requires GOOS=js GOARCH=wasm")
	}
	return nil
}

func PreJSPath(llgoRoot string) string {
	return filepath.Join(llgoRoot, "internal", "wasmworkers", "worker_pre.js")
}
