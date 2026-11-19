package build

import (
	"fmt"
	"go/types"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/binaryspecialize"
	"github.com/xgo-dev/llgo/internal/build/jsonspecialize"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

func specializeBinaryCodecs(conf *Config, frontend cl.Options, goroot string, stdlib *packages.Package, initial, runtimePkgs []*packages.Package, shortBuffer string) (map[*types.Package]bool, error) {
	return specializeSizeCodec(conf, frontend, goroot, stdlib, initial, runtimePkgs,
		func(closure []*packages.Package, library, entry *packages.Package) ([]*packages.Package, error) {
			return binaryspecialize.Rewrite(closure, library, entry, shortBuffer)
		}, "fixed-size binary encoding")
}

func specializeJSONMarshals(conf *Config, frontend cl.Options, goroot string, stdlib *packages.Package, initial, runtimePkgs []*packages.Package) (map[*types.Package]bool, error) {
	return specializeSizeCodec(conf, frontend, goroot, stdlib, initial, runtimePkgs, jsonspecialize.Rewrite, "statically typed JSON encoding")
}

// The binary accessor belongs to a fixed compiler profile and is installed
// before collecting callers. The cache key depends only on its presence.
func prepareBinaryCodecSupport(conf *Config, frontend cl.Options, goroot string, library *packages.Package) (string, error) {
	if conf.disableSizeSpecialization || !sizeCodecBuild(conf, frontend) || library == nil || library.TypesInfo == nil || !isStandardLibraryPackage(library, goroot) {
		return "", nil
	}
	return binaryspecialize.AddShortBufferAccessor(library)
}

func sizeCodecBuild(conf *Config, frontend cl.Options) bool {
	return conf.OptLevel == optlevel.Oz && conf.ltoMode() == lto.Full && conf.deadcodeDropEnabled() &&
		conf.BuildMode == BuildModeExe && conf.Mode != ModeGen && conf.Coverage == nil &&
		conf.ModuleHook == nil && !frontend.Debug && !frontend.DebuggerFrames &&
		!conf.LTOPlugin.Enabled() && !frontendCacheEnabled(conf) && len(conf.SourceOverlay) == 0
}

func specializeSizeCodec(conf *Config, frontend cl.Options, goroot string, stdlib *packages.Package, initial, runtimePkgs []*packages.Package, rewrite func([]*packages.Package, *packages.Package, *packages.Package) ([]*packages.Package, error), description string) (map[*types.Package]bool, error) {
	if conf.disableSizeSpecialization || !sizeCodecBuild(conf, frontend) || len(initial) != 1 || initial[0].Name != "main" {
		return nil, nil
	}
	if stdlib == nil || stdlib.TypesInfo == nil || !isStandardLibraryPackage(stdlib, goroot) {
		return nil, nil
	}
	var closure []*packages.Package
	var aliased bool
	packages.Visit(slices.Concat(initial, runtimePkgs), func(p *packages.Package) bool {
		for _, f := range p.Syntax {
			for _, group := range f.Comments {
				for _, item := range directive.ParseGroup(group) {
					if item.Name != "go:linkname" && item.Name != "llgo:link" {
						continue
					}
					args := strings.Fields(item.Args)
					for _, arg := range args {
						aliased = aliased || strings.HasPrefix(arg, stdlib.PkgPath+".")
					}
				}
			}
		}
		closure = append(closure, p)
		return !aliased
	}, nil)
	if aliased {
		return nil, nil
	}
	changed, err := rewrite(closure, stdlib, initial[0])
	if err != nil || len(changed) == 0 {
		return nil, err
	}
	rewritten := make(map[*types.Package]bool, len(changed))
	for _, pkg := range changed {
		rewritten[pkg.Types] = true
	}
	if conf.Verbose || conf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: specialize "+description)
	}
	return rewritten, nil
}
