package targets

import "github.com/xgo-dev/llgo/target"

// Loader handles loading and parsing target configurations.
type Loader = target.Loader

// NewLoader creates a new target configuration loader.
func NewLoader(targetsDir string) *Loader {
	return target.NewLoader(targetsDir)
}
