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

package target

import "fmt"

// Resolver provides a high-level interface for resolving and validating target
// configurations from a targets directory.
type Resolver struct {
	loader *Loader
}

// NewResolver creates a target resolver rooted at targetsDir.
func NewResolver(targetsDir string) *Resolver {
	return &Resolver{
		loader: NewLoader(targetsDir),
	}
}

// Resolve resolves and validates a target configuration by name.
func (r *Resolver) Resolve(targetName string) (*Config, error) {
	config, err := r.loader.Load(targetName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target %s: %w", targetName, err)
	}
	if err := r.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid target config %s: %w", targetName, err)
	}
	return config, nil
}

// ResolveAll resolves all available configurations for inspection. Use Resolve
// to validate a selected target before building it.
func (r *Resolver) ResolveAll() (map[string]*Config, error) {
	return r.loader.LoadAll()
}

// ListAvailableTargets returns all available target names in the directory.
func (r *Resolver) ListAvailableTargets() ([]string, error) {
	return r.loader.ListTargets()
}

// validateConfig validates that a resolved config has required fields and valid
// WebAssembly memory/stack settings.
func (r *Resolver) validateConfig(config *Config) error {
	if config.Name == "" {
		return fmt.Errorf("target name is required")
	}

	if config.WasmInitialMemory != 0 || config.WasmMaxMemory != 0 || config.WasmStackSize != 0 {
		if config.GOOS != "wasip1" || config.GOARCH != "wasm" || config.WasmProfile != "w32" || config.WasmProvider != "wasi" {
			return fmt.Errorf("wasm memory/stack settings require the w32/wasi profile")
		}
		for _, field := range []struct {
			name  string
			value uint64
		}{
			{"wasm-initial-memory", config.WasmInitialMemory},
			{"wasm-max-memory", config.WasmMaxMemory},
		} {
			if field.value > 1<<32 || field.value%65536 != 0 {
				return fmt.Errorf("%s must be a multiple of 65536 bytes and at most 4 GiB", field.name)
			}
		}
		// Memory can span 4 GiB, but a wasm32 stack pointer cannot hold 2^32.
		if config.WasmStackSize >= 1<<32 || config.WasmStackSize%16 != 0 {
			return fmt.Errorf("wasm-stack-size must be a multiple of 16 bytes and below 4 GiB")
		}
		// Compare effective initial/maximum sizes in wasm-ld, after explicit
		// linker flags and response files have overridden inherited defaults.
	}

	return nil
}

// GetTargetsDirectory returns the path to the targets directory.
func (r *Resolver) GetTargetsDirectory() string {
	return r.loader.GetTargetsDir()
}

// HasTarget reports whether a target with the given name exists.
func (r *Resolver) HasTarget(name string) bool {
	_, err := r.loader.LoadRaw(name)
	return err == nil
}
