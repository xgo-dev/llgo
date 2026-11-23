package calltemplate

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/ssa"
)

type formatRule struct {
	format int
	args   int
	c      bool
	result string
}

func bindFormat(fn *types.Func, d directive.Directive, doc *ast.CommentGroup, trustedFmt bool) (formatRule, error) {
	c := d.Name == "llgointernal:template:cfmt"
	r := formatRule{format: -1, args: -1, c: c}
	sig := fn.Signature()
	for _, binding := range strings.Fields(d.Args) {
		name, value, ok := strings.Cut(binding, "=")
		index := paramIndex(sig, value)
		switch {
		case ok && index >= 0 && name == "format" && r.format < 0:
			r.format = index
		case ok && index >= 0 && name == "args" && r.args < 0:
			r.args = index
		default:
			return r, fmt.Errorf("invalid format template binding %q", binding)
		}
	}
	if sig.Recv() != nil || !sig.Variadic() || r.args != sig.Params().Len()-1 || r.format < 0 || r.format >= r.args {
		return r, fmt.Errorf("format template requires a fixed format parameter and a variadic argument parameter")
	}
	tail := sig.Params().At(r.args).Type().Underlying().(*types.Slice)
	if !types.Identical(tail.Elem(), types.Universe.Lookup("any").Type()) {
		return r, fmt.Errorf("format template requires ...any")
	}
	switch typ := sig.Params().At(r.format).Type().Underlying().(type) {
	case *types.Basic:
		if c || typ.Kind() != types.String {
			return r, fmt.Errorf("gofmt requires a string format parameter")
		}
	case *types.Pointer:
		if basic, ok := typ.Elem().Underlying().(*types.Basic); !c || !ok || (basic.Kind() != types.Int8 && basic.Kind() != types.Uint8) {
			return r, fmt.Errorf("cfmt requires a C character pointer")
		}
	default:
		return r, fmt.Errorf("invalid format parameter type")
	}
	if c {
		link := externalName(doc)
		if !ssa.HasNameValist(sig) || r.format != cFormatIndex(link) || r.args != r.format+1 {
			return r, fmt.Errorf("cfmt requires a C printf-family declaration with %s", ssa.NameValist)
		}
	} else {
		if !trustedFmt || ssa.HasNameValist(sig) {
			return r, fmt.Errorf("gofmt requires a supported standard fmt entry")
		}
		r.result = goFormatResult(fn.Name(), r)
		if r.result == "" {
			return r, fmt.Errorf("unsupported gofmt output for %s", fn.Name())
		}
	}
	return r, nil
}

func cFormatIndex(name string) int {
	switch name {
	case "C.printf":
		return 0
	case "C.fprintf", "C.sprintf":
		return 1
	case "C.snprintf":
		return 2
	}
	return -1
}

func externalName(doc *ast.CommentGroup) string {
	for _, d := range directive.ParseGroup(doc) {
		if d.Name == "go:linkname" || d.Name == "llgo:link" {
			_, target := directive.ParseLinknameArgs(d.Args)
			return target
		}
	}
	return ""
}

// The supported standard entries have the same rules as explicit directives.
// Their declarations come from the selected Go source release.
func (r rules) addStandardFormats(pkg *packages.Package) {
	for _, name := range []string{"Printf", "Sprintf", "Appendf", "Fprintf"} {
		fn, _ := pkg.Types.Scope().Lookup(name).(*types.Func)
		if fn == nil {
			continue
		}
		if _, exists := r.formats[fn]; exists {
			continue
		}
		sig := fn.Signature()
		f, args := paramIndex(sig, "format"), sig.Params().Len()-1
		if f < 0 || f >= args || !sig.Variadic() || !types.Identical(sig.Params().At(f).Type(), types.Typ[types.String]) {
			continue
		}
		rule := formatRule{format: f, args: args}
		rule.result = goFormatResult(name, rule)
		if rule.result != "" {
			r.formats[fn] = rule
		}
	}
}

func goFormatResult(name string, rule formatRule) string {
	switch {
	case rule.args == 1 && name == "Printf":
		return "stdout"
	case rule.args == 1 && name == "Sprintf":
		return "string"
	case rule.args == 2 && name == "Appendf":
		return "append"
	case rule.args == 2 && name == "Fprintf":
		return "file"
	}
	return ""
}

type formatPart struct {
	text  string
	raw   string
	verb  byte
	width int
	zero  bool
}

// parseFormat accepts sequential operands, escaped percent signs, and literal
// widths. It leaves argument selection, types and dialect checks to consumers.
func parseFormat(text string) ([]formatPart, bool) {
	var parts []formatPart
	for i := 0; i < len(text); {
		end := strings.IndexByte(text[i:], '%')
		if end < 0 {
			parts = append(parts, formatPart{text: text[i:]})
			break
		}
		if end != 0 {
			parts = append(parts, formatPart{text: text[i : i+end]})
			i += end
		}
		start := i
		i++
		if i == len(text) {
			return nil, false
		}
		if text[i] == '%' {
			parts = append(parts, formatPart{text: "%", raw: "%%"})
			i++
			continue
		}
		part := formatPart{}
		if text[i] == '0' {
			part.zero = true
			i++
		}
		for i < len(text) && '0' <= text[i] && text[i] <= '9' {
			part.width = part.width*10 + int(text[i]-'0')
			if part.width > 99 {
				return nil, false
			}
			i++
		}
		if i >= len(text) || !(text[i] >= 'a' && text[i] <= 'z' || text[i] >= 'A' && text[i] <= 'Z') {
			return nil, false
		}
		part.verb, part.raw = text[i], text[start:i+1]
		parts = append(parts, part)
		i++
	}
	return parts, true
}

func constantString(info *types.Info, expr ast.Expr) (string, bool) {
	value := info.Types[expr].Value
	if value == nil || value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(value), true
}

func astString(info *types.Info, old ast.Expr, value string) ast.Expr {
	lit := &ast.BasicLit{ValuePos: old.Pos(), Kind: token.STRING, Value: fmt.Sprintf("%q", value)}
	tv := info.Types[old]
	tv.Value = constant.MakeString(value)
	info.Types[lit] = tv
	return lit
}
