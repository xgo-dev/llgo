package calltemplate

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"
	"strings"
)

func asCString(info *types.Info, expr ast.Expr, cstrings map[*types.Func]bool) (*ast.CallExpr, string, bool) {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !cstrings[calledFunc(info, call.Fun)] {
		return nil, "", false
	}
	text, ok := constantString(info, call.Args[0])
	return call, text, ok
}

// Keep the same libc entry and fold only formats whose constant substitutions
// preserve the observed byte stream. In particular this keeps snprintf's
// truncation, return count and buffer ownership in libc.
func foldCFormat(info *types.Info, call *ast.CallExpr, rule formatRule, cstrings map[*types.Func]bool) bool {
	if len(call.Args) < rule.args {
		return false
	}
	format, text, ok := asCString(info, call.Args[rule.format], cstrings)
	if !ok || strings.ContainsRune(text, 0) {
		return false
	}
	parts, ok := parseFormat(text)
	if !ok {
		return false
	}
	args := append([]ast.Expr(nil), call.Args[:rule.args]...)
	index := rule.args
	var result strings.Builder
	changed := false
	for _, part := range parts {
		if part.verb == 0 {
			if part.raw == "%%" {
				result.WriteString(part.raw)
			} else {
				result.WriteString(part.text)
			}
			continue
		}
		if index >= len(call.Args) || !strings.ContainsRune("diouxXfFeEgGaAcsp", rune(part.verb)) {
			return false
		}
		arg := call.Args[index]
		index++
		value, ok := cConstant(info, arg, part, cstrings)
		if !ok {
			args = append(args, arg)
			result.WriteString(part.raw)
			continue
		}
		result.WriteString(strings.ReplaceAll(value, "%", "%%"))
		changed = true
	}
	if !changed || index != len(call.Args) {
		return false
	}
	format.Args[0] = astString(info, format.Args[0], result.String())
	call.Args = args
	return true
}

func cConstant(info *types.Info, arg ast.Expr, part formatPart, cstrings map[*types.Func]bool) (string, bool) {
	if part.width != 0 || part.zero {
		return "", false
	}
	if part.verb == 's' {
		_, text, ok := asCString(info, arg, cstrings)
		text, _, _ = strings.Cut(text, "\x00")
		return text, ok
	}
	tv := info.Types[arg]
	if tv.Type == nil || tv.Value == nil || tv.Value.Kind() != constant.Int {
		return "", false
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	if !ok {
		return "", false
	}
	switch part.verb {
	case 'd', 'i':
		if b.Kind() == types.Int32 {
			value, ok := constant.Int64Val(tv.Value)
			return strconv.FormatInt(value, 10), ok
		}
	case 'u', 'o', 'x', 'X':
		if b.Kind() == types.Uint32 {
			value, ok := constant.Uint64Val(tv.Value)
			base := 10
			if part.verb == 'o' {
				base = 8
			} else if part.verb == 'x' || part.verb == 'X' {
				base = 16
			}
			text := strconv.FormatUint(value, base)
			if part.verb == 'X' {
				text = strings.ToUpper(text)
			}
			return text, ok
		}
	}
	return "", false
}
