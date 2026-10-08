package build

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
)

// These settings affect package code, not just the final link. Independent
// programs may share a backend/cache entry only when these settings agree.
type initialBuildFeatures struct {
	// Used to group compatible initials; each child re-derives its active
	// localities from its own package subset instead of overriding the registry.
	localContext    bool
	reflectBridges  bool
	funcInfoEntries bool
}

type initialBuildGroup struct {
	features initialBuildFeatures
	pkgs     []*packages.Package
}

// A plan crosses the analysis lifetime boundary without retaining its graph.
// Trace ownership moves to Build so its parent span still encloses the children.
type initialBuildPlan struct {
	invocations []Invocation
	finishTrace func()
}

func groupInitialBuilds(ctx *context, alts []*packages.Package) []initialBuildGroup {
	if len(ctx.initial) < 2 || ctx.mode == ModeGen {
		return nil
	}
	var groups []initialBuildGroup
	indexes := make(map[initialBuildFeatures]int)
	target := ctx.prog.Target()
	for _, pkg := range ctx.initial {
		features := initialBuildFeatures{
			localContext: ctx.prog.NeedsLocalContextForPackages(activeLocalityPackages([]*packages.Package{pkg}, alts)),
		}
		if target.GOARCH == "wasm" {
			use := analyzeWasmInitialUse(ctx.progSSA, pkg.Types)
			features.reflectBridges = (target.WasmProvider == "wasi" || target.GOOS == "js") && use.usesWasmReflectBridges()
			features.funcInfoEntries = ctx.buildConf.BuildMode != BuildModeExe || use.usesRuntimeFuncForPC()
		}
		index, ok := indexes[features]
		if !ok {
			index = len(groups)
			indexes[features] = index
			groups = append(groups, initialBuildGroup{features: features})
		}
		groups[index].pkgs = append(groups[index].pkgs, pkg)
	}
	return groups
}

func initialGroupInvocations(inv Invocation, ctx *context, groups []initialBuildGroup) []Invocation {
	children := make([]Invocation, 0, len(groups))
	for _, group := range groups {
		conf := ctx.buildConf.clone()
		args := make([]string, len(group.pkgs))
		for i, pkg := range group.pkgs {
			args[i] = initialGroupLoadArg(ctx, pkg)
		}
		// Preserve invocation controls. Normal grouping starts before fallback;
		// fallback builds one root at a time and therefore cannot enter grouping.
		child := inv
		child.Args, child.Config, child.Dir = args, conf, ctx.commands.dir
		child.multipleInitials = true
		// Taking &group.features would retain group.pkgs and their entire
		// frontend graph even after the parent build frame has returned.
		features := group.features
		child.initialFeatures = &features
		child.parentBuildTrace = ctx.buildTrace
		children = append(children, child)
	}
	return children
}

func initialGroupLoadArg(ctx *context, pkg *packages.Package) string {
	path := pkg.PkgPath
	if ctx.mode == ModeTest {
		path = strings.TrimSuffix(path, ".test")
	}
	if pkg.Dir == "" || (!strings.HasPrefix(path, "_/") && path != "command-line-arguments") {
		return path
	}
	rel, err := filepath.Rel(ctx.commands.dir, pkg.Dir)
	if err != nil {
		return path
	}
	if rel == "." {
		return rel
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

func buildInitialGroups(children []Invocation) ([]Package, error) {
	var result []Package
	var failures []error
	// Keep the original package-worker budget. Ordinary test batches generally
	// form a single group and retain the shared frontend and native test DAG.
	for _, child := range children {
		pkgs, err := Build(child)
		result = append(result, pkgs...)
		if err != nil {
			failures = append(failures, err)
		}
	}
	return result, errors.Join(failures...)
}
