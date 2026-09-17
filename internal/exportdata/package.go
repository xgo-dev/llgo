// Package exportdata describes LLGo declaration properties persisted in package manifests.
package exportdata

import (
	"fmt"
	"sort"
)

const Version = 1

// Package is immutable once associated with a loaded package. An empty Functions
// list with a supported Version is distinct from a missing exports section.
type Package struct {
	Version   int        `yaml:"version"`
	Functions []Function `yaml:"functions"`
}

// Function identifies a source declaration, not its resolved linker symbol.
// Name is package-relative: F, T.Method, or (*T).Method. Receiver aliases retain
// their declared spelling. Instances use the generic origin's declaration.
type Function struct {
	Name     string `yaml:"name"`
	Cold     bool   `yaml:"cold,omitempty"`
	NoReturn bool   `yaml:"noreturn,omitempty"`
}

func (p *Package) Validate() error {
	if p == nil {
		return fmt.Errorf("missing package exports")
	}
	if p.Version != Version {
		return fmt.Errorf("unsupported package exports version %d", p.Version)
	}
	for i, f := range p.Functions {
		if f.Name == "" {
			return fmt.Errorf("empty exported declaration name")
		}
		if i > 0 && p.Functions[i-1].Name >= f.Name {
			return fmt.Errorf("unordered or duplicate exported declaration %q", f.Name)
		}
	}
	return nil
}

func (p *Package) Function(name string) Function {
	if p == nil {
		return Function{}
	}
	i := sort.Search(len(p.Functions), func(i int) bool { return p.Functions[i].Name >= name })
	if i < len(p.Functions) && p.Functions[i].Name == name {
		return p.Functions[i]
	}
	return Function{}
}
