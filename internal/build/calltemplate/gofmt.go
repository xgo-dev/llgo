package calltemplate

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
)

type formatCall struct {
	call *ast.CallExpr
	name string
}

type formatSource struct {
	pkg     *packages.Package
	library *packages.Package
	used    map[*ast.Ident]bool
	helpers map[string]string
	body    strings.Builder
	calls   []formatCall
	next    int
	alias   string
	ioAlias string
}

func newFormatSource(pkg, library *packages.Package) *formatSource {
	return &formatSource{pkg: pkg, library: library, used: make(map[*ast.Ident]bool), helpers: make(map[string]string)}
}

func (s *formatSource) name() string {
	for {
		s.next++
		n := fmt.Sprintf("__llgo_format_template_%d", s.next)
		if s.pkg.Types.Scope().Lookup(n) == nil {
			return n
		}
	}
}

func (s *formatSource) plan(call *ast.CallExpr, rule formatRule) {
	if s.library == nil || len(call.Args) < rule.args ||
		s.library.Types.Scope().Lookup("LLGoTemplateStdout") == nil {
		return
	}
	if rule.result == "file" && !s.fileWriter(call.Args[1-rule.format]) {
		return
	}
	text, ok := constantString(s.pkg.TypesInfo, call.Args[rule.format])
	if !ok {
		return
	}
	parts, ok := ParseFormat(text)
	if !ok || len(parts) > 64 {
		return
	}
	var operands []*types.Basic
	for _, arg := range call.Args[rule.args:] {
		typ := s.pkg.TypesInfo.TypeOf(arg)
		if typ == nil {
			return
		}
		basic, ok := types.Unalias(types.Default(typ)).(*types.Basic)
		if !ok || s.pkg.Types.Scope().Lookup(basic.Name()) != nil {
			return
		}
		operands = append(operands, basic)
	}
	// The literal optimization is already handled in the SSA formatter.
	if len(operands) == 0 {
		return
	}
	ops := make([]string, 0, len(operands))
	for i := range parts {
		part := &parts[i]
		if part.Verb == 0 {
			continue
		}
		if len(ops) >= len(operands) {
			return
		}
		op := formatOperand(part, operands[len(ops)])
		if op == "" {
			return
		}
		ops = append(ops, op)
	}
	if len(ops) != len(operands) {
		return
	}
	for _, name := range []string{"append", "byte", "bool", "int", "uint64", "string", "error", "nil"} {
		if s.pkg.Types.Scope().Lookup(name) != nil {
			return
		}
	}
	var key strings.Builder
	fmt.Fprintf(&key, "%s:%d:%q", rule.result, rule.format, text)
	for _, b := range operands {
		fmt.Fprintf(&key, "/%s", b.Name())
	}
	name := s.helpers[key.String()]
	if name == "" {
		name = s.name()
		s.helpers[key.String()] = name
		if s.alias == "" {
			s.alias = s.name()
		}
		s.emit(name, rule, parts, operands, ops)
	}
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		s.used[f] = true
	case *ast.SelectorExpr:
		s.used[f.Sel] = true
	}
	s.calls = append(s.calls, formatCall{call, name})
}

func formatOperand(part *FormatPart, arg *types.Basic) string {
	switch {
	case arg.Kind() == types.String:
		if part.Verb == 'v' {
			part.Verb = 's'
		}
		if part.Verb == 's' || part.Verb == 'q' || part.Verb == 'x' || part.Verb == 'X' {
			return "String"
		}
	case arg.Kind() == types.Bool:
		if part.Verb == 'v' {
			part.Verb = 't'
		}
		if part.Verb == 't' {
			return "Bool"
		}
	case arg.Info()&types.IsInteger != 0:
		if part.Verb == 'v' {
			part.Verb = 'd'
		}
		if strings.ContainsRune("dboxX", rune(part.Verb)) {
			return "Integer"
		}
	}
	return ""
}

func (s *formatSource) emit(name string, rule formatRule, parts []FormatPart, operands []*types.Basic, ops []string) {
	b := &s.body
	fmt.Fprintf(b, "func %s(", name)
	for i := 0; i < rule.args; i++ {
		if i == rule.format {
			fmt.Fprint(b, "_ string,")
		} else if rule.result == "file" {
			if s.ioAlias == "" {
				s.ioAlias = s.name()
			}
			fmt.Fprintf(b, "w %s.Writer,", s.ioAlias)
		} else {
			fmt.Fprint(b, "dst []byte,")
		}
	}
	for i, typ := range operands {
		fmt.Fprintf(b, "x%d %s,", i, typ.Name())
	}
	switch rule.result {
	case "append":
		fmt.Fprintln(b, ") []byte {\nvar buf [64]byte; out := buf[:0]")
	case "string":
		fmt.Fprintln(b, ") string {\nvar buf [64]byte; out := buf[:0]")
	case "stdout", "file":
		fmt.Fprintln(b, ") (int, error) {\nvar buf [64]byte; out := buf[:0]")
	}
	i := 0
	for _, part := range parts {
		if part.Verb == 0 {
			fmt.Fprintf(b, "out = append(out,%q...)\n", part.Text)
			continue
		}
		arg := fmt.Sprintf("x%d", i)
		if ops[i] == "Integer" {
			arg = fmt.Sprintf("uint64(%s), %t", arg, operands[i].Info()&types.IsUnsigned == 0)
		}
		fmt.Fprintf(b, "out = %s.LLGoTemplate%s(out,%s,%q,%d,%t)\n", s.alias, ops[i], arg, part.Verb, part.Width, part.Zero)
		i++
	}
	switch rule.result {
	case "append":
		fmt.Fprintln(b, "return append(dst, out...)\n}")
	case "string":
		fmt.Fprintln(b, "return string(out)\n}")
	case "stdout":
		fmt.Fprintf(b, "return %s.LLGoTemplateStdout(out)\n}\n", s.alias)
	case "file":
		fmt.Fprintln(b, "return w.Write(out)\n}")
	}
}

func (s *formatSource) apply() ([]string, error) {
	if len(s.calls) == 0 {
		return nil, nil
	}
	// The size profile avoids adding an emitter alongside a generic formatter
	// still used by the same package. Existing fmt passes handle mixed calls.
	for id, obj := range s.pkg.TypesInfo.Uses {
		if fn, ok := obj.(*types.Func); ok && fn.Pkg() == s.library.Types && fn.Exported() && !s.used[id] {
			return nil, nil
		}
	}
	var src strings.Builder
	fmt.Fprintf(&src, "package %s\nimport %s %q\n", s.pkg.Name, s.alias, "fmt")
	imports := typesynth.Importer{"fmt": s.library.Types}
	if s.ioAlias != "" {
		io := s.library.Imports["io"]
		fmt.Fprintf(&src, "import %s %q\n", s.ioAlias, "io")
		imports["io"] = io.Types
		s.pkg.Imports["io"] = io
	}
	src.WriteString(s.body.String())
	if err := typesynth.AddFile(s.pkg, "__llgo_format_template.go", src.String(), imports, ""); err != nil {
		return nil, err
	}
	var changes []string
	for _, site := range s.calls {
		fn := s.pkg.Types.Scope().Lookup(site.name).(*types.Func)
		typesynth.BindCall(s.pkg.TypesInfo, site.call, fn, types.Instance{Type: fn.Type()})
		changes = append(changes, changeKey(s.pkg, site.call, "gofmt", site.name))
	}
	return changes, nil
}

// An arbitrary Writer can observe fmt's call frames. Select the direct path
// only for a known standard *os.File.
func (s *formatSource) fileWriter(arg ast.Expr) bool {
	os := s.library.Imports["os"]
	io := s.library.Imports["io"]
	if os == nil || os.Types == nil || io == nil || io.Types == nil {
		return false
	}
	file := os.Types.Scope().Lookup("File")
	return file != nil && types.Identical(s.pkg.TypesInfo.TypeOf(arg), types.NewPointer(file.Type()))
}
