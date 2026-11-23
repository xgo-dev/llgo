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
	formats map[*types.Func]formatRule
	fmt     *packages.Package
	value   *types.Func
	codecs  []*codecBindings
}

// NeedsFormatRuntime reports whether a declared template can emit composite
// formatters with calls to the shared formatting runtime.
func (b *Bindings) NeedsFormatRuntime() bool {
	return b != nil && b.value != nil && b.formats[b.value].composite
}

type codecBindings struct {
	library *packages.Package
	dialect string
	entries map[string]string
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
	var codecs *codecBindings
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
				name, rest, ok := takeWord(d.Args)
				dialect, args, _ := takeWord(rest)
				fn, _ := pkg.Types.Scope().Lookup(name).(*types.Func)
				if !ok || fn == nil {
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
				case "binary", "json":
					op, err := bindCodec(fn.Signature(), dialect, args)
					if err != nil {
						return fmt.Errorf("%s: %w", pkg.Fset.Position(d.Pos), err)
					}
					if codecs == nil {
						codecs = &codecBindings{library: pkg, dialect: dialect, entries: make(map[string]string)}
					}
					if _, duplicate := codecs.entries[name]; duplicate || codecs.dialect != dialect {
						return fmt.Errorf("%s: conflicting codec binding for %s", pkg.Fset.Position(d.Pos), name)
					}
					codecs.entries[name] = op
				default:
					return fmt.Errorf("%s: unknown template dialect %q", pkg.Fset.Position(d.Pos), dialect)
				}
			}
		}
	}
	if codecs != nil {
		b.codecs = append(b.codecs, codecs)
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

// Validate the operand roles used by the semantic generators. The emitter
// retains the declaration's evaluation order and original result types.
func bindCodec(sig *types.Signature, dialect, args string) (string, error) {
	opts, err := bindingOptions(args)
	if err != nil {
		return "", err
	}
	op := opts["op"]
	delete(opts, "op")
	var roles []string
	var results []types.Type
	errorType := types.Universe.Lookup("error").Type()
	bytesType := types.NewSlice(types.Typ[types.Byte])
	switch dialect + "/" + op {
	case "binary/fallback", "json/fallback":
	case "binary/size":
		roles, results = []string{"value"}, []types.Type{types.Typ[types.Int]}
	case "binary/encode", "binary/decode":
		roles, results = []string{"buffer", "order", "value"}, []types.Type{types.Typ[types.Int], errorType}
	case "json/marshal":
		roles, results = []string{"value"}, []types.Type{bytesType, errorType}
	case "json/unmarshal":
		roles, results = []string{"input", "value"}, []types.Type{errorType}
	default:
		return "", fmt.Errorf("unknown %s template operation %q", dialect, op)
	}
	if len(opts) != len(roles) || (op != "fallback" && (sig.Recv() != nil || sig.Variadic() ||
		sig.Params().Len() != len(roles) || sig.Results().Len() != len(results))) {
		return "", fmt.Errorf("invalid signature or roles for %s %s template", dialect, op)
	}
	for i, role := range roles {
		p := sig.Params().At(i)
		if p.Name() != opts[role] || (role == "value" && !types.IsInterface(p.Type())) ||
			((role == "buffer" || role == "input") && !types.Identical(p.Type(), bytesType)) {
			return "", fmt.Errorf("invalid %s role for %s %s template", role, dialect, op)
		}
	}
	for i, result := range results {
		if !types.Identical(sig.Results().At(i).Type(), result) {
			return "", fmt.Errorf("invalid result type for %s %s template", dialect, op)
		}
	}
	return op, nil
}
