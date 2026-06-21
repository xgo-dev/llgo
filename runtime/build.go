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
	"internal/abi":            {mode: altPkgReplace},
	"internal/runtime/atomic": {mode: altPkgReplace, goarchs: map[string]struct{}{"arm": {}}},
	"internal/reflectlite":    {mode: altPkgReplace},
	"internal/runtime/maps":   {mode: altPkgReplace},
	"internal/runtime/sys":    {mode: altPkgAdditive},
	"reflect":                 {mode: altPkgReplace},
	"runtime":                 {mode: altPkgReplace},
	"sync/atomic":             {mode: altPkgReplace},
	"syscall/js":              {mode: altPkgReplace},
	"syscall":                 {mode: altPkgReplace},
	"unique":                  {mode: altPkgReplace},
	"golang.org/x/sys/unix":   {mode: altPkgReplace},
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

var sourcePatchPkgs = map[string]struct{}{
	"crypto/internal/constanttime": {},
	"internal/sync":                {},
	"iter":                         {},
	"runtime/metrics":              {},
}
