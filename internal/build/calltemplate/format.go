package calltemplate

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/ssa"
)

type formatRule struct {
	format, args             int
	c                        bool
	result                   string
	literal, scalar, dynamic string
	composite                bool
}

func bindFormat(fn *types.Func, d directive.Directive, doc *ast.CommentGroup, trustedFmt bool) (formatRule, error) {
	c := d.Name == "llgointernal:template:cfmt"
	r := formatRule{format: -1, args: -1, c: c}
	opts, err := bindingOptions(d.Args)
	if err != nil {
		return r, err
	}
	sig := fn.Signature()
	for name, value := range opts {
		switch name {
		case "format":
			r.format = paramIndex(sig, value)
			if r.format < 0 {
				return r, fmt.Errorf("unknown format parameter %q", value)
			}
		case "args":
			r.args = paramIndex(sig, value)
		case "result", "literal", "scalar", "dynamic", "composite":
			if c {
				return r, fmt.Errorf("invalid format template binding %q", name)
			}
		default:
			return r, fmt.Errorf("invalid format template binding %q", name)
		}
	}
	if sig.Recv() != nil || !sig.Variadic() || r.args < 0 || r.args != sig.Params().Len()-1 || r.format >= r.args {
		return r, fmt.Errorf("format template requires a fixed format parameter and a variadic argument parameter")
	}
	tail := sig.Params().At(r.args).Type().Underlying().(*types.Slice)
	if !types.Identical(tail.Elem(), types.Universe.Lookup("any").Type()) {
		return r, fmt.Errorf("format template requires ...any")
	}
	if c {
		if !ssa.HasNameValist(sig) || r.format < 0 || r.format != cFormatIndex(externalName(doc)) || r.args != r.format+1 {
			return r, fmt.Errorf("cfmt requires a C printf-family declaration with %s", ssa.NameValist)
		}
		ptr, ok := sig.Params().At(r.format).Type().Underlying().(*types.Pointer)
		if ok {
			basic, ok := ptr.Elem().Underlying().(*types.Basic)
			if ok && (basic.Kind() == types.Int8 || basic.Kind() == types.Uint8) {
				return r, nil
			}
		}
		return r, fmt.Errorf("cfmt requires a C character pointer")
	}
	if !trustedFmt || ssa.HasNameValist(sig) {
		return r, fmt.Errorf("gofmt requires a supported standard fmt entry")
	}
	r.result, r.literal, r.scalar, r.dynamic = opts["result"], opts["literal"], opts["scalar"], opts["dynamic"]
	r.composite = opts["composite"] == "true"
	if opts["composite"] != "" && !r.composite {
		return r, fmt.Errorf("invalid composite format binding")
	}
	if r.format < 0 {
		if r.args == 0 && r.scalar != "" && len(opts) == 2 {
			return r, nil
		}
		return r, fmt.Errorf("gofmt requires a format parameter or a line formatter")
	}
	if !types.Identical(sig.Params().At(r.format).Type(), types.Typ[types.String]) {
		return r, fmt.Errorf("gofmt requires a string format parameter")
	}
	result := goFormatResult(sig, r)
	if result == "" || r.result != "" && r.result != result {
		return r, fmt.Errorf("unsupported gofmt output for %s", fn.Name())
	}
	r.result = result
	if (r.literal != "" || r.scalar != "" || r.dynamic != "" || r.composite) && (r.format != 0 || r.args != 1) {
		return r, fmt.Errorf("invalid helper binding for %s", fn.Name())
	}
	if (r.dynamic != "" || r.composite) && r.result != "string" {
		return r, fmt.Errorf("value formatting requires a string result")
	}
	return r, nil
}

func goFormatResult(sig *types.Signature, r formatRule) string {
	out := sig.Results()
	if out.Len() == 2 && types.Identical(out.At(0).Type(), types.Typ[types.Int]) &&
		types.Identical(out.At(1).Type(), types.Universe.Lookup("error").Type()) {
		switch r.args {
		case 1:
			return "stdout"
		case 2:
			return "file"
		}
	}
	if out.Len() == 1 {
		b := types.NewSlice(types.Typ[types.Byte])
		switch {
		case r.args == 1 && types.Identical(out.At(0).Type(), types.Typ[types.String]):
			return "string"
		case r.args == 2 && types.Identical(sig.Params().At(1-r.format).Type(), b) &&
			types.Identical(out.At(0).Type(), b):
			return "append"
		}
	}
	return ""
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
