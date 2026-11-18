package build

import (
	"fmt"
	"go/types"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/build/binaryspecialize"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
)

func specializeBinaryCodecs(conf *Config, frontend cl.Options, goroot string, stdlib *packages.Package, initial, runtimePkgs []*packages.Package) (map[*types.Package]bool, error) {
	if conf.OptLevel != optlevel.Oz || conf.ltoMode() != lto.Full || !conf.deadcodeDropEnabled() ||
		conf.BuildMode != BuildModeExe || conf.Mode == ModeGen || conf.Coverage != nil ||
		conf.ModuleHook != nil || frontend.Debug || frontend.DebuggerFrames ||
		conf.LTOPlugin.Enabled() || frontendCacheEnabled(conf) || len(conf.SourceOverlay) != 0 || len(initial) != 1 {
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
						aliased = aliased || strings.HasPrefix(arg, "encoding/binary.")
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
	changed, err := binaryspecialize.Rewrite(closure, stdlib)
	if err != nil || len(changed) == 0 {
		return nil, err
	}
	rewritten := make(map[*types.Package]bool, len(changed))
	for _, pkg := range changed {
		rewritten[pkg.Types] = true
	}
	if conf.Verbose || conf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: specialize fixed-size binary encoding")
	}
	return rewritten, nil
}
