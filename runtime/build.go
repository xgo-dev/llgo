package runtime

import "sort"

type altPkgSpec struct {
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

var altPkgs = map[string]altPkgSpec{
	"internal/abi":         {},
	"internal/reflectlite": {},
	"reflect":              {},
	"runtime":              {},
	"syscall/js":           {},
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

// SourcePatchReplacesAsm checks the policy for an active source patch.
func SourcePatchReplacesAsm(path, goarch string, tags []string) bool {
	switch path {
	case "internal/bytealg", "internal/chacha8rand",
		"runtime/internal/syscall", "internal/runtime/syscall", "internal/runtime/syscall/linux":
		for _, tag := range tags {
			if tag == "baremetal" {
				return true
			}
		}
		return false
	default:
		return SourcePatchReplacesAsmForGOARCH(path, goarch)
	}
}

var sourcePatchPkgs = map[string]struct{}{
	"crypto/internal/fips140/aes":    {},
	"crypto/internal/constanttime":   {},
	"crypto/internal/sysrand":        {},
	"fmt":                            {},
	"go/build":                       {},
	"go/importer":                    {},
	"internal/bytealg":               {},
	"internal/chacha8rand":           {},
	"internal/cpu":                   {},
	"internal/runtime/atomic":        {},
	"internal/runtime/maps":          {},
	"internal/runtime/syscall":       {},
	"internal/runtime/syscall/linux": {},
	"internal/runtime/sys":           {},
	"internal/sync":                  {},
	"iter":                           {},
	"math/bits":                      {},
	"os":                             {},
	"reflect":                        {},
	"runtime":                        {},
	"runtime/internal/atomic":        {},
	"runtime/internal/syscall":       {},
	"runtime/metrics":                {},
	"simd":                           {},
	"simd/internal/bridge":           {},
	"sync":                           {},
	"sync/atomic":                    {},
	"syscall":                        {},
	"syscall/js":                     {},
	"unique":                         {},
}

var sourcePatchAsmPkgs = map[string]map[string]struct{}{
	"crypto/internal/fips140/aes": {"amd64": {}, "386": {}, "arm64": {}, "arm": {}, "ppc64": {}, "ppc64le": {}, "riscv64": {}},
	"internal/runtime/atomic":     {"arm": {}},
	"runtime/internal/atomic":     {"arm": {}},
	"sync/atomic":                 {"*": {}},
	"syscall":                     {"*": {}},
}
