package build

import (
	"fmt"
	"go/types"
	"os"
	"slices"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/calltemplate"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

func loadSourceBindings(conf *Config, frontend cl.Options, goroot string, initial []*packages.Package) (*calltemplate.Bindings, error) {
	if conf.OptLevel == optlevel.O0 || frontend.DebuggerFrames {
		return nil, nil
	}
	return calltemplate.LoadBindings(initial, func(p *packages.Package) bool {
		if !isStandardLibraryPackage(p, goroot) {
			return false
		}
		if len(conf.SourceOverlay) != 0 {
			for _, name := range slices.Concat(p.GoFiles, p.CompiledGoFiles) {
				if _, replaced := conf.SourceOverlay[name]; replaced {
					return false
				}
			}
		}
		return true
	})
}

func applySourceTemplates(conf *Config, frontend cl.Options, bindings *calltemplate.Bindings, initial []*packages.Package, verbose bool) (map[*types.Package][]string, error) {
	if bindings == nil || frontendCacheEnabled(conf) {
		return nil, nil
	}
	formats := !conf.disableSizeSpecialization && sizeCodecBuild(conf, frontend)
	selected, err := bindings.Rewrite(initial, formats)
	if verbose {
		for _, changes := range selected {
			for _, change := range changes {
				fmt.Fprintln(os.Stderr, "llgo: apply call template", change)
			}
		}
	}
	return selected, err
}
