package runtime

import "sort"

type altPkgMode uint8

const (
	altPkgReplace altPkgMode = iota + 1
	altPkgAdditive
)

type altPkgSpec struct {
	mode    altPkgMode
	goarchs map[string]struct{}
}

func (s altPkgSpec) enabledFor(goarch string) bool {
	return len(s.goarchs) == 0 || hasGoarch(s.goarchs, goarch)
}

func hasGoarch(goarchs map[string]struct{}, goarch string) bool {
	if goarchs == nil {
		return false
	}
	_, ok := goarchs[goarch]
	return ok
}

func SkipToBuild(pkgPath string) bool {
	if _, ok := altPkgs[pkgPath]; ok {
		return false
	}
	return pkgPath == "unsafe"
}

func HasAltPkg(path string) (b bool) {
	_, b = altPkgs[path]
	return
}

func HasAltPkgForGOARCH(path, goarch string) bool {
	spec, ok := altPkgs[path]
	return ok && spec.enabledFor(goarch)
}

func HasAdditiveAltPkg(path string) bool {
	return altPkgs[path].mode == altPkgAdditive
}

func HasAdditiveAltPkgForGOARCH(path, goarch string) bool {
	spec, ok := altPkgs[path]
	return ok && spec.mode == altPkgAdditive && spec.enabledFor(goarch)
}

var altPkgs = map[string]altPkgSpec{
	"internal/abi":         {mode: altPkgReplace},
	"internal/reflectlite": {mode: altPkgReplace},
	"reflect":              {mode: altPkgReplace},
	"runtime":              {mode: altPkgReplace},
	"syscall/js":           {mode: altPkgReplace},
}

func HasSourcePatchPkg(path string) bool {
	_, ok := sourcePatchPkgs[path]
	return ok
}

func SourcePatchPkgPaths() []string {
	paths := make([]string, 0, len(sourcePatchPkgs))
	for path := range sourcePatchPkgs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func SourcePatchReplacesAsmForGOARCH(path, goarch string) bool {
	goarchs, ok := sourcePatchAsmPkgs[path]
	return ok && (hasGoarch(goarchs, "*") || hasGoarch(goarchs, goarch))
}

var sourcePatchPkgs = map[string]struct{}{
	"crypto/internal/constanttime": {},
	"internal/bytealg":             {},
	"internal/chacha8rand":         {},
	"internal/runtime/atomic":      {},
	"internal/runtime/maps":        {},
	"internal/runtime/sys":         {},
	"internal/sync":                {},
	"iter":                         {},
	"runtime":                      {},
	"runtime/metrics":              {},
	"sync":                         {},
	"sync/atomic":                  {},
	"syscall":                      {},
	"unique":                       {},
}

var sourcePatchAsmPkgs = map[string]map[string]struct{}{
	"internal/bytealg":        {"wasm": {}},
	"internal/chacha8rand":    {"wasm": {}},
	"internal/runtime/atomic": {"arm": {}, "wasm": {}},
	"sync/atomic":             {"*": {}},
	"syscall":                 {"*": {}},
}
