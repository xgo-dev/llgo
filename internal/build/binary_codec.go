package build

import (
	"fmt"
	"go/types"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/binaryspecialize"
	"github.com/xgo-dev/llgo/internal/build/calltemplate"
	"github.com/xgo-dev/llgo/internal/build/jsonspecialize"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

type sourceTemplates struct {
	binaryLibrary *types.Package
	binary        map[*types.Package]bool
	json          map[*types.Package]bool
	calls         map[*types.Package][]string
}

func applySourceTemplates(conf *Config, frontend cl.Options, goroot string, dedup packages.Deduper, initial, runtimePkgs []*packages.Package, verbose bool) (sourceTemplates, error) {
	var result sourceTemplates
	if conf.OptLevel == optlevel.O0 || frontend.DebuggerFrames || frontendCacheEnabled(conf) {
		return result, nil
	}
	library := func(path string) *packages.Package {
		p := dedup.Check(path)
		if p == nil || p.TypesInfo == nil || !isStandardLibraryPackage(p.Package, goroot) {
			return nil
		}
		return p.Package
	}
	formats := !conf.disableSizeSpecialization && sizeCodecBuild(conf, frontend)
	if formats {
		binary := library("encoding/binary")
		shortBuffer, err := prepareBinaryCodecSupport(conf, frontend, goroot, binary)
		if err != nil {
			return result, fmt.Errorf("%w: %w", errGeneratedSizeSource, err)
		}
		if shortBuffer != "" {
			result.binaryLibrary = binary.Types
		}
		result.binary, result.json, err = rewriteClosedCodecs(initial, runtimePkgs, binary, library("encoding/json"), shortBuffer, verbose)
		if err != nil {
			return result, fmt.Errorf("%w: %w", errGeneratedSizeSource, err)
		}
	}
	calls, err := calltemplate.Rewrite(initial, library("fmt"), formats)
	result.calls = calls
	if verbose {
		for _, changes := range calls {
			for _, change := range changes {
				fmt.Fprintln(os.Stderr, "llgo: apply call template", change)
			}
		}
	}
	return result, err
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

// The decoder helpers are a fixed build profile. Final-link initialization
// pruning still requires the normal proof for the selected target.
func typedJSONSupport(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return !conf.disableSizeSpecialization && sizeCodecBuild(conf, frontend) &&
		(embeddedSizePruningConfig(conf, target, frontend) ||
			wasiSizePruningConfig(conf, target, frontend) ||
			hostedValueFormatSupport(conf, target, frontend) ||
			linuxSizeCodecBuild(conf, target, frontend))
}

func linuxSizeCodecBuild(conf *Config, target crosscompile.Export, frontend cl.Options) bool {
	return !conf.disableSizeSpecialization && conf.Target == "" && conf.Goos == "linux" &&
		(conf.Goarch == "amd64" || conf.Goarch == "arm64") &&
		len(target.ExtraFiles) == 0 && conf.LinkOptions.ExternalLinker == "" &&
		conf.LinkOptions.ExternalLinkerFlags == "" && sizeCodecBuild(conf, frontend)
}

// Both source codecs require a closed entry package. Resolve linker aliases
// once before handing the same package graph to the codec emitters.
func rewriteClosedCodecs(initial, runtimePkgs []*packages.Package, binary, json *packages.Package, shortBuffer string, verbose bool) (map[*types.Package]bool, map[*types.Package]bool, error) {
	if len(initial) != 1 || initial[0].Name != "main" || binary == nil && json == nil {
		return nil, nil, nil
	}
	var closure []*packages.Package
	aliases := make(map[string]bool)
	packages.Visit(slices.Concat(initial, runtimePkgs), func(p *packages.Package) bool {
		for _, f := range p.Syntax {
			for _, comments := range f.Comments {
				for _, item := range directive.ParseGroup(comments) {
					if item.Name == "go:linkname" || item.Name == "llgo:link" {
						for _, arg := range strings.Fields(item.Args) {
							for _, path := range []string{"encoding/binary", "encoding/json"} {
								if strings.HasPrefix(arg, path+".") {
									aliases[path] = true
								}
							}
						}
					}
				}
			}
		}
		closure = append(closure, p)
		return true
	}, nil)
	var binaryPackages, jsonPackages map[*types.Package]bool
	if binary != nil && !aliases[binary.PkgPath] {
		changed, err := binaryspecialize.Rewrite(closure, binary, initial[0], shortBuffer)
		if err != nil {
			return nil, nil, err
		}
		binaryPackages = recordCodecRewrites(changed, "fixed-size binary encoding", verbose)
	}
	if json != nil && !aliases[json.PkgPath] {
		changed, err := jsonspecialize.Rewrite(closure, json, initial[0])
		if err != nil {
			return nil, nil, err
		}
		jsonPackages = recordCodecRewrites(changed, "statically typed JSON codec", verbose)
	}
	return binaryPackages, jsonPackages, nil
}

func recordCodecRewrites(changed []*packages.Package, description string, verbose bool) map[*types.Package]bool {
	if len(changed) == 0 {
		return nil
	}
	rewritten := make(map[*types.Package]bool, len(changed))
	for _, pkg := range changed {
		rewritten[pkg.Types] = true
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "llgo: specialize "+description)
	}
	return rewritten
}
