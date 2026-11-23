package build

import (
	"fmt"
	"go/types"
	"os"
	"slices"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/calltemplate"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

type sourceTemplates struct {
	calltemplate.CodecUse
	calls map[*types.Package][]string
}

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

func applySourceTemplates(conf *Config, frontend cl.Options, bindings *calltemplate.Bindings, initial, runtimePkgs []*packages.Package, verbose bool) (sourceTemplates, error) {
	var result sourceTemplates
	if bindings == nil || frontendCacheEnabled(conf) {
		return result, nil
	}
	var err error
	formats := !conf.disableSizeSpecialization && sizeCodecBuild(conf, frontend)
	if formats {
		result.CodecUse, err = bindings.RewriteCodecs(initial, runtimePkgs)
		if err != nil {
			return result, fmt.Errorf("%w: %w", errGeneratedSizeSource, err)
		}
	}
	result.calls, err = bindings.Rewrite(initial, formats)
	if verbose {
		if len(result.Binary) != 0 {
			fmt.Fprintln(os.Stderr, "llgo: specialize fixed-size binary encoding")
		}
		if len(result.JSON) != 0 {
			fmt.Fprintln(os.Stderr, "llgo: specialize statically typed JSON codec")
		}
		for _, changes := range result.calls {
			for _, change := range changes {
				fmt.Fprintln(os.Stderr, "llgo: apply call template", change)
			}
		}
	}
	return result, err
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
