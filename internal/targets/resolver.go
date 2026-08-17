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

// ResolveAll resolves all available target configurations
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

	// For now, we don't require any specific fields since different targets
	// may have different requirements. This can be extended in the future.

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
