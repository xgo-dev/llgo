package build

import (
	"go/ast"
	"go/build"

	"github.com/xgo-dev/llgo/internal/build/sourcepatch"
)

func cloneOverlay(src map[string][]byte) map[string][]byte {
	return sourcepatch.CloneOverlay(src)
}

type sourcePatchBuildContext struct {
	goos       string
	goarch     string
	goversion  string
	toolTags   []string
	buildFlags []string
	targetTags []string
	cgoEnabled *bool
}

func (c sourcePatchBuildContext) toPackage() sourcepatch.BuildContext {
	return sourcepatch.BuildContext{
		GOOS:       c.goos,
		GOARCH:     c.goarch,
		GoVersion:  c.goversion,
		ToolTags:   c.toolTags,
		BuildFlags: c.buildFlags,
		TargetTags: c.targetTags,
		CgoEnabled: c.cgoEnabled,
	}
}

func buildSourcePatchOverlayForGOROOT(base map[string][]byte, runtimeDir, goroot string, ctx sourcePatchBuildContext) (map[string][]byte, map[string][]string, error) {
	return sourcepatch.BuildOverlayForGOROOT(base, runtimeDir, goroot, ctx.toPackage())
}

func applySourcePatchForPkg(base, current map[string][]byte, runtimeDir, goroot, pkgPath string, ctx sourcePatchBuildContext) (bool, map[string][]byte, []string, error) {
	return sourcepatch.ApplyForPkg(base, current, runtimeDir, goroot, pkgPath, ctx.toPackage())
}

func sourcePatchMayContainSkip(src []byte, skips map[string]struct{}) bool {
	return sourcepatch.MayContainSkip(src, skips)
}

func newSourcePatchMatchContext(goroot string, ctx sourcePatchBuildContext) (build.Context, error) {
	return sourcepatch.NewMatchContext(goroot, ctx.toPackage())
}

// sourcePatchBuildTags is shared by the overlay and the assembly planner.
func sourcePatchBuildTags(buildFlags, targetTags []string) []string {
	return sourcepatch.BuildTags(buildFlags, targetTags)
}

func parseSourcePatchBuildTags(buildFlags []string) []string {
	return sourcepatch.ParseBuildTags(buildFlags)
}

func splitSourcePatchBuildTags(s string) []string {
	return sourcepatch.SplitBuildTags(s)
}

func parseSourcePatchDirective(line string) (skipAll bool, names []string, ok bool) {
	return sourcepatch.ParseDirective(line)
}

func buildInjectedSourcePatchFile(filename string, src []byte) []byte {
	return sourcepatch.BuildInjectedFile(filename, src)
}

func filterSourcePatchFile(src []byte, skips map[string]struct{}) ([]byte, bool, error) {
	return sourcepatch.FilterFile(src, skips)
}

func declPatchKeys(decl ast.Decl) []string {
	return sourcepatch.DeclKeys(decl)
}

func recvPatchKey(expr ast.Expr) string {
	return sourcepatch.RecvKey(expr)
}
