package fmtspecialize

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

// ValueFunctions identifies the concrete helpers generated for one package.
type ValueFunctions struct {
	values *typeutil.Map
	stdout map[token.Pos]string
}

func (g *formatSource) addStdout(info *types.Info, entry string, call *ast.CallExpr, helpers map[string]*typeutil.Map) string {
	args := call.Args
	key := entry
	var parts []string
	var verbs []byte
	if entry == "Printf" {
		if len(args) < 2 {
			return ""
		}
		format := info.Types[args[0]].Value
		if format == nil || format.Kind() != constant.String {
			return ""
		}
		text := constant.StringVal(format)
		key = entry + "\x00" + text
		parts, verbs = defaultFormatParts(text)
		args = args[1:]
	} else {
		if len(args) == 0 {
			return ""
		}
		parts = make([]string, len(args)+1)
		for i := 1; i < len(args); i++ {
			parts[i] = " "
		}
		parts[len(args)] = "\n"
	}
	if len(parts) != len(args)+1 {
		return ""
	}

	params := make([]*types.Var, len(args))
	for i, arg := range args {
		t := info.TypeOf(arg)
		if t == nil || llssa.HasTypeParam(t) {
			return ""
		}
		t = types.Default(t)
		// Capture of the stdout file remains unobservable when formatting
		// cannot call a method on an operand or any of its fields.
		if len(verbs) != 0 && !matchesDefaultVerb(t, verbs[i]) ||
			!defaultFormatType(t, true, make(map[types.Type]bool), nil) {
			return ""
		}
		params[i] = types.NewVar(token.NoPos, nil, fmt.Sprintf("v%d", i), t)
	}
	tuple := types.NewTuple(params...)
	matches := helpers[key]
	if matches == nil {
		matches = new(typeutil.Map)
		helpers[key] = matches
	}
	if name, ok := matches.At(tuple).(string); ok {
		return name
	}

	name := g.nextName()
	matches.Set(tuple, name)
	var signature, body strings.Builder
	for i, param := range params {
		if i > 0 {
			signature.WriteString(", ")
		}
		fmt.Fprintf(&signature, "%s %s", param.Name(), g.typeName(param.Type()))
		appendFormatLiteral(&body, parts[i])
		fmt.Fprintf(&body, "dst = %s(dst, %s)\n", g.appendFunc(param.Type(), true), param.Name())
	}
	appendFormatLiteral(&body, parts[len(parts)-1])
	fmt.Fprintf(&g.funcs, "func %s(%s) (int, error) {\nvar dst []byte\n%sreturn __llgo_std_fmt.LlgoPrintfValue(string(dst))\n}\n",
		name, &signature, &body)
	return name
}

func appendFormatLiteral(body *strings.Builder, text string) {
	if text != "" {
		fmt.Fprintf(body, "dst = append(dst, %q...)\n", text)
	}
}

// A few unmodified scalar verbs use the same formatting as %v.
func matchesDefaultVerb(t types.Type, verb byte) bool {
	if verb == 'v' {
		return true
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	if verb == 'g' {
		return basic.Info()&(types.IsFloat|types.IsComplex) != 0
	}
	return scalarDefaultVerb(basic.Kind()) == verb
}

// Split a format around supported verbs and decode literal percent signs.
func defaultFormatParts(format string) (parts []string, verbs []byte) {
	var literal strings.Builder
	for len(format) != 0 {
		text, rest, hasVerb := strings.Cut(format, "%")
		literal.WriteString(text)
		if !hasVerb {
			break
		}
		if rest == "" {
			return nil, nil
		}
		switch rest[0] {
		case '%':
			literal.WriteByte('%')
		case 'v', 'd', 's', 't', 'g':
			verbs = append(verbs, rest[0])
			parts = append(parts, literal.String())
			literal.Reset()
		default:
			return nil, nil
		}
		format = rest[1:]
	}
	return append(parts, literal.String()), verbs
}
