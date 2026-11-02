package targets

import (
	"fmt"
	"path/filepath"

	"github.com/xgo-dev/llgo/internal/env"
)

// Resolver provides high-level interface for target configuration resolution
type Resolver struct {
	loader *Loader
}

// NewResolver creates a new target resolver
func NewResolver(targetsDir string) *Resolver {
	return &Resolver{
		loader: NewLoader(targetsDir),
	}
}

// NewDefaultResolver creates a resolver with default targets directory
func NewDefaultResolver() *Resolver {
	llgoRoot := env.LLGoROOT()
	targetsDir := filepath.Join(llgoRoot, "targets")

	return NewResolver(targetsDir)
}

// Resolve resolves a target configuration by name
func (r *Resolver) Resolve(targetName string) (*Config, error) {
	config, err := r.loader.Load(targetName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target %s: %w", targetName, err)
	}

	// Validate required fields
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

// ListAvailableTargets returns a list of all available target names
func (r *Resolver) ListAvailableTargets() ([]string, error) {
	return r.loader.ListTargets()
}

// validateConfig validates that a resolved config has required fields
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

// GetTargetsDirectory returns the path to the targets directory
func (r *Resolver) GetTargetsDirectory() string {
	return r.loader.GetTargetsDir()
}

// HasTarget checks if a target with the given name exists
func (r *Resolver) HasTarget(name string) bool {
	_, err := r.loader.LoadRaw(name)
	return err == nil
}
