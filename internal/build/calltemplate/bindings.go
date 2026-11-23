package calltemplate

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
)

// Bindings carries the trusted library templates from source through SSA.
type Bindings struct {
	formats    map[*types.Func]formatRule
	fmt        *packages.Package
	value      *types.Func
	structural []*structuralBindings
}

// NeedsFormatRuntime reports whether a declared template can emit composite
// formatters with calls to the shared formatting runtime.
func (b *Bindings) NeedsFormatRuntime() bool {
	return b != nil && b.value != nil && b.formats[b.value].composite
}

// LoadBindings reads declarations shipped in the selected standard-library
// patches. Keeping the bindings alongside helpers avoids copies of upstream
// declarations and leaves the original Go implementation available.
func LoadBindings(roots []*packages.Package, trusted func(*packages.Package) bool) (*Bindings, error) {
	b := &Bindings{formats: make(map[*types.Func]formatRule)}
	var err error
	packages.Visit(roots, func(pkg *packages.Package) bool {
		if err == nil && pkg.Types != nil && trusted(pkg) {
			err = b.loadPackage(pkg)
		}
		return err == nil
	}, nil)
	return b, err
}

func (b *Bindings) loadPackage(pkg *packages.Package) error {
	var structural *structuralBindings
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if !strings.HasPrefix(comment.Text, "//llgointernal:template:bind") {
					continue
				}
				d, ok := directive.Parse(comment)
				if !ok || d.Name != "llgointernal:template:bind" {
					continue
				}
				name, rest, _ := takeWord(d.Args)
				dialect, args, _ := takeWord(rest)
				fn, _ := pkg.Types.Scope().Lookup(name).(*types.Func)
				if fn == nil {
					return fmt.Errorf("%s: invalid template entry %q", pkg.Fset.Position(d.Pos), name)
				}
				switch dialect {
				case "gofmt":
					d.Name, d.Args = "llgointernal:template:gofmt", args
					rule, err := bindFormat(fn, d, nil, true)
					if err != nil {
						return fmt.Errorf("%s: %w", pkg.Fset.Position(d.Pos), err)
					}
					if _, duplicate := b.formats[fn]; duplicate || b.fmt != nil && b.fmt != pkg {
						return fmt.Errorf("%s: duplicate Go format binding", pkg.Fset.Position(d.Pos))
					}
					b.formats[fn], b.fmt = rule, pkg
					if rule.composite || rule.dynamic != "" {
						if b.value != nil {
							return fmt.Errorf("%s: duplicate value format binding", pkg.Fset.Position(d.Pos))
						}
						b.value = fn
					}
				case "any":
					if structural == nil {
						structural = &structuralBindings{library: pkg, entries: make(map[string]structuralRule)}
					}
					if err := structural.bind(fn, args); err != nil {
						return fmt.Errorf("%s: %w", pkg.Fset.Position(d.Pos), err)
					}
				default:
					return fmt.Errorf("%s: unknown template dialect %q", pkg.Fset.Position(d.Pos), dialect)
				}
			}
		}
	}
	if structural != nil {
		b.structural = append(b.structural, structural)
	}
	return nil
}

func takeWord(text string) (word, rest string, ok bool) {
	text = strings.TrimSpace(text)
	if i := strings.IndexAny(text, " \t"); i >= 0 {
		return text[:i], strings.TrimSpace(text[i:]), true
	}
	return text, "", false
}

func bindingOptions(args string) (map[string]string, error) {
	opts := make(map[string]string)
	for _, field := range strings.Fields(args) {
		name, value, ok := strings.Cut(field, "=")
		if !ok || name == "" || value == "" || opts[name] != "" {
			return nil, fmt.Errorf("invalid template binding %q", field)
		}
		opts[name] = value
	}
	return opts, nil
}
