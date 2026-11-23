package calltemplate

import (
	"go/types"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
)

type CodecUse struct {
	Binary, JSON map[*types.Package]bool
}

// RewriteCodecs plans each declared codec before changing any call of that
// codec. Unknown entry points keep the generic implementation.
func (b *Bindings) RewriteCodecs(roots, runtimePkgs []*packages.Package) (CodecUse, error) {
	var result CodecUse
	if len(roots) != 1 || roots[0].Name != "main" || len(b.codecs) == 0 {
		return result, nil
	}
	var closure []*packages.Package
	aliases := make(map[string]bool)
	paths := make([]string, 0, len(b.codecs))
	for _, codec := range b.codecs {
		paths = append(paths, codec.library.PkgPath+".")
	}
	visit := func(pkg *packages.Package) bool {
		closure = append(closure, pkg)
		for _, file := range pkg.Syntax {
			for _, comments := range file.Comments {
				for _, d := range directive.ParseGroup(comments) {
					if d.Name != "go:linkname" && d.Name != "llgo:link" {
						continue
					}
					for _, arg := range strings.Fields(d.Args) {
						for _, prefix := range paths {
							if strings.HasPrefix(arg, prefix) {
								aliases[prefix] = true
							}
						}
					}
				}
			}
		}
		return true
	}
	packages.Visit(slices.Concat(roots, runtimePkgs), visit, nil)
	for _, codec := range b.codecs {
		if aliases[codec.library.PkgPath+"."] {
			continue
		}
		var changed bool
		var err error
		switch codec.dialect {
		case "binary":
			changed, err = rewriteBinary(closure, codec, roots[0], shortBufferHelper(codec.library.Types))
			if changed {
				result.Binary = map[*types.Package]bool{roots[0].Types: true}
			}
		case "json":
			changed, err = rewriteJSON(closure, codec, roots[0])
			if changed {
				result.JSON = map[*types.Package]bool{roots[0].Types: true}
			}
		}
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (b *codecBindings) hasEntry(name string) bool { return b.entries[name] != "" }

// A compact codec is profitable when every use of its declared entries is
// matched in the executable package. Reusable package archives stay intact.
func planClosedCodec[P any](closure []*packages.Package, b *codecBindings, entry *packages.Package,
	plan func(*packages.Package) (P, bool, bool)) (P, bool) {
	var selected P
	var found bool
	for _, pkg := range closure {
		lib := b.library
		if pkg == lib || pkg.Imports[lib.PkgPath] == nil {
			continue
		}
		if pkg.Imports[lib.PkgPath].Types != lib.Types || pkg.TypesInfo == nil {
			return selected, false
		}
		p, matched, valid := plan(pkg)
		if !valid || matched && pkg != entry {
			return selected, false
		}
		if matched {
			selected, found = p, true
		}
	}
	return selected, found
}
