package calltemplate

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
)

type cFormatSource struct {
	info     *types.Info
	cstrings map[*types.Func]bool
	uses     map[string]int
}

func newCFormatSource(pkg *packages.Package, cstrings map[*types.Func]bool) cFormatSource {
	s := cFormatSource{info: pkg.TypesInfo, cstrings: cstrings}
	if pkg.Name != "main" || len(cstrings) == 0 {
		return s
	}
	s.uses = make(map[string]int)
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if _, text, ok := s.asCString(call); ok {
					s.uses[text]++
				}
			}
			return true
		})
	}
	return s
}

func (s cFormatSource) asCString(expr ast.Expr) (*ast.CallExpr, string, bool) {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !s.cstrings[calledFunc(s.info, call.Fun)] {
		return nil, "", false
	}
	text, ok := constantString(s.info, call.Args[0])
	return call, text, ok
}

// Keep the same libc entry and fold only formats whose constant substitutions
// preserve the observed byte stream. In particular this keeps snprintf's
// truncation, return count and buffer ownership in libc.
func (s cFormatSource) fold(call *ast.CallExpr, rule formatRule) bool {
	if len(call.Args) < rule.args {
		return false
	}
	format, text, ok := s.asCString(call.Args[rule.format])
	if !ok || strings.ContainsRune(text, 0) {
		return false
	}
	parts, ok := ParseFormat(text)
	if !ok {
		return false
	}
	args := append([]ast.Expr(nil), call.Args[:rule.args]...)
	index := rule.args
	var result strings.Builder
	changed := false
	for _, part := range parts {
		if part.Verb == 0 {
			if part.Raw == "%%" {
				result.WriteString(part.Raw)
			} else {
				result.WriteString(part.Text)
			}
			continue
		}
		if index >= len(call.Args) || !strings.ContainsRune("diouxXfFeEgGaAcsp", rune(part.Verb)) {
			return false
		}
		arg := call.Args[index]
		index++
		value, ok := s.constant(arg, part)
		if !ok {
			args = append(args, arg)
			result.WriteString(part.Raw)
			continue
		}
		result.WriteString(value)
		changed = true
	}
	if !changed || index != len(call.Args) || s.uses[text] > 1 && result.Len() > len(text) {
		return false
	}
	format.Args[0] = astString(s.info, format.Args[0], result.String())
	call.Args = args
	return true
}

func (s cFormatSource) constant(arg ast.Expr, part FormatPart) (string, bool) {
	if part.Width != 0 || part.Zero {
		return "", false
	}
	if part.Verb == 's' {
		_, text, ok := s.asCString(arg)
		value, _, _ := strings.Cut(text, "\x00")
		value = strings.ReplaceAll(value, "%", "%%")
		// A shared literal is already deduplicated. Leave it in the
		// argument list when expanding it would grow each format string.
		return value, ok && (s.uses[text] < 2 || len(value) <= len(part.Raw))
	}
	tv := s.info.Types[arg]
	if tv.Type == nil || tv.Value == nil || tv.Value.Kind() != constant.Int {
		return "", false
	}
	b, ok := tv.Type.Underlying().(*types.Basic)
	if !ok {
		return "", false
	}
	switch part.Verb {
	case 'd', 'i':
		if b.Kind() == types.Int32 {
			value, ok := constant.Int64Val(tv.Value)
			return strconv.FormatInt(value, 10), ok
		}
	case 'u', 'o', 'x', 'X':
		if b.Kind() == types.Uint32 {
			value, ok := constant.Uint64Val(tv.Value)
			base := 10
			if part.Verb == 'o' {
				base = 8
			} else if part.Verb == 'x' || part.Verb == 'X' {
				base = 16
			}
			text := strconv.FormatUint(value, base)
			if part.Verb == 'X' {
				text = strings.ToUpper(text)
			}
			return text, ok
		}
	}
	return "", false
}
