package jsonspecialize

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
)

type emitter struct {
	pkg         *types.Package
	body        strings.Builder
	next        int
	encoders    map[*shape]string
	strconv     string
	sort        string
	appendQuote string
}

func (p packagePlan) apply(jsonV2 bool, sortPkg, strconvPkg, utf8Pkg *packages.Package) error {
	e := &emitter{pkg: p.pkg.Types, encoders: make(map[*shape]string)}
	quote, intAlias, sortAlias, utf8Alias := e.name(), e.name(), e.name(), e.name()
	e.strconv, e.sort, e.appendQuote = intAlias, sortAlias+".Strings", quote
	if sortPkg.PkgPath == "slices" {
		e.sort = sortAlias + ".Sort"
	}
	wrappers := make(map[*shape]string)
	for _, site := range p.calls {
		s := site.shape
		if wrappers[s] != "" {
			continue
		}
		fn := e.emit(s)
		wrappers[s] = e.name()
		typ := e.alias(s.typ)
		fmt.Fprintf(&e.body, "func %s(v %s) ([]byte,error) { return %s(nil,v),nil }\n", wrappers[s], typ, fn)
	}
	var source strings.Builder
	fmt.Fprintf(&source, "package %s\n", p.pkg.Name)
	imports := make(typesynth.Importer)
	if strings.Contains(e.body.String(), quote+"(") {
		// The v2-backed encoder writes the replacement rune directly. The
		// original encoder writes the escaped form for invalid UTF-8.
		replacement := "\\ufffd"
		if jsonV2 {
			replacement = "\uFFFD"
		}
		fmt.Fprintf(&e.body, "func %s%s\n", quote, fmt.Sprintf(quoteBody, replacement, utf8Alias))
	}
	for _, item := range []struct {
		pkg   *packages.Package
		alias string
	}{{utf8Pkg, utf8Alias}, {strconvPkg, intAlias}, {sortPkg, sortAlias}} {
		if strings.Contains(e.body.String(), item.alias+".") {
			fmt.Fprintf(&source, "import %s %q\n", item.alias, item.pkg.PkgPath)
			imports[item.pkg.PkgPath] = item.pkg.Types
			p.pkg.Imports[item.pkg.PkgPath] = item.pkg
		}
	}
	source.WriteString(e.body.String())
	if err := typesynth.AddFile(p.pkg, "__llgo_json.go", source.String(), imports, ""); err != nil {
		return err
	}
	for _, site := range p.calls {
		name := wrappers[site.shape]
		fn := p.pkg.Types.Scope().Lookup(name).(*types.Func)
		old := site.call.Fun
		id := &ast.Ident{Name: name, NamePos: old.Pos()}
		tv := p.pkg.TypesInfo.Types[ast.Unparen(old)]
		tv.Type = fn.Type()
		p.pkg.TypesInfo.Uses[id] = fn
		p.pkg.TypesInfo.Types[id] = tv
		site.call.Fun = id
	}
	return nil
}

func (e *emitter) name() string {
	for {
		e.next++
		name := fmt.Sprintf("__llgo_json_%d", e.next)
		if e.pkg.Scope().Lookup(name) == nil {
			return name
		}
	}
}

func (e *emitter) alias(typ types.Type) string {
	name := e.name()
	obj := types.NewTypeName(token.NoPos, e.pkg, name, nil)
	types.NewAlias(obj, typ)
	e.pkg.Scope().Insert(obj)
	return name
}

func (e *emitter) emit(s *shape) string {
	if name := e.encoders[s]; name != "" {
		return name
	}
	name := e.name()
	e.encoders[s] = name
	var element string
	if s.elem != nil {
		element = e.emit(s.elem)
	}
	fields := make([]string, len(s.fields))
	for i, member := range s.fields {
		fields[i] = e.emit(member.value)
	}
	b := &e.body
	fmt.Fprintf(b, "func %s(dst []byte,v %s) []byte {\n", name, e.alias(s.typ))
	switch t := s.typ.Underlying().(type) {
	case *types.Basic:
		switch {
		case t.Info()&types.IsBoolean != 0:
			fmt.Fprintln(b, "if v { return append(dst,\"true\"...) }; return append(dst,\"false\"...)")
		case t.Info()&types.IsString != 0:
			fmt.Fprintf(b, "return %s(dst,string(v))\n", e.appendQuote)
		case t.Info()&types.IsUnsigned != 0:
			fmt.Fprintf(b, "return %s.AppendUint(dst,uint64(v),10)\n", e.strconv)
		default:
			fmt.Fprintf(b, "return %s.AppendInt(dst,int64(v),10)\n", e.strconv)
		}
	case *types.Pointer:
		fmt.Fprintf(b, "if v==nil { return append(dst,\"null\"...) }; return %s(dst,*v)\n", element)
	case *types.Array, *types.Slice:
		if _, ok := t.(*types.Slice); ok {
			fmt.Fprintln(b, "if v==nil { return append(dst,\"null\"...) }")
		}
		fmt.Fprintf(b, "dst=append(dst,'['); for i,x:=range v { if i!=0 { dst=append(dst,',') }; dst=%s(dst,x) }; return append(dst,']')\n", element)
	case *types.Map:
		fmt.Fprintln(b, "if v==nil { return append(dst,\"null\"...) }\nkeys:=make([]string,0,len(v)); for k:=range v { keys=append(keys,k) }")
		fmt.Fprintf(b, "%s(keys)\ndst=append(dst,'{'); for i,k:=range keys { if i!=0 { dst=append(dst,',') }; dst=%s(dst,k); dst=append(dst,':'); dst=%s(dst,v[k]) }; return append(dst,'}')\n",
			e.sort, e.appendQuote, element)
	case *types.Struct:
		fmt.Fprintln(b, "dst=append(dst,'{')")
		if len(s.fields) != 0 {
			fmt.Fprintln(b, "start:=len(dst)")
		}
		for i, member := range s.fields {
			expr := "v." + member.goName
			cond := notEmpty(member, expr)
			if cond != "" {
				fmt.Fprintf(b, "if %s {\n", cond)
			}
			key, _ := json.Marshal(member.name)
			fmt.Fprintf(b, "if len(dst)!=start { dst=append(dst,',') }; dst=append(dst,%q...); dst=%s(dst,%s)\n", string(key)+":", fields[i], expr)
			if cond != "" {
				fmt.Fprintln(b, "}")
			}
		}
		fmt.Fprintln(b, "return append(dst,'}')")
	}
	fmt.Fprintln(b, "}")
	return name
}

func notEmpty(member field, value string) string {
	if !member.omit {
		return ""
	}
	switch t := member.value.typ.Underlying().(type) {
	case *types.Array, *types.Slice, *types.Map:
		return "len(" + value + ")!=0"
	case *types.Pointer:
		return value + "!=nil"
	case *types.Basic:
		switch {
		case t.Info()&types.IsBoolean != 0:
			return value
		case t.Info()&types.IsString != 0:
			return value + "!=\"\""
		default:
			return value + "!=0"
		}
	}
	return ""
}

const quoteBody = `(dst []byte, s string) []byte {
	const hex = "0123456789abcdef"
	dst = append(dst, '"')
	for i := 0; i < len(s); {
		b := s[i]
		if b < %[2]s.RuneSelf {
			i++
			switch b {
			case '"', '\\':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				if b < 0x20 || b == '<' || b == '>' || b == '&' {
					dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&15])
				} else {
					dst = append(dst, b)
				}
			}
			continue
		}
		r, n := %[2]s.DecodeRuneInString(s[i:])
		switch {
		case r == %[2]s.RuneError && n == 1:
			dst = append(dst, %[1]q...)
		case r == '\u2028' || r == '\u2029':
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[r&15])
		default:
			dst = append(dst, s[i:i+n]...)
		}
		i += n
	}
	return append(dst, '"')
}
`
