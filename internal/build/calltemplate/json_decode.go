package calltemplate

import (
	"fmt"
	"go/types"
)

func canDecode(s *shape) bool {
	for _, t := range []types.Type{s.typ, types.NewPointer(s.typ)} {
		methods := types.NewMethodSet(t)
		for i := range methods.Len() {
			switch methods.At(i).Obj().Name() {
			case "UnmarshalJSON", "UnmarshalJSONFrom", "UnmarshalText":
				return false
			}
		}
	}
	if s.elem != nil && !canDecode(s.elem) {
		return false
	}
	for _, field := range s.fields {
		if !canDecode(field.value) {
			return false
		}
	}
	return true
}

func (e *emitter) emitDecode(s *shape) string {
	if fn := e.decoders[s]; fn != "" {
		return fn
	}
	name := e.name()
	e.decoders[s] = name
	var element string
	if s.elem != nil {
		element = e.emitDecode(s.elem)
	}
	fields := make([]string, len(s.fields))
	for i, f := range s.fields {
		fields[i] = e.emitDecode(f.value)
	}
	b := &e.body
	typ := e.alias(s.typ)
	fmt.Fprintf(b, "func %s(d *%s.Decoder,v *%s) error {\n", name, e.jsonText, typ)
	switch t := s.typ.Underlying().(type) {
	case *types.Basic:
		helper := "LlgoReadInt"
		switch {
		case t.Info()&types.IsBoolean != 0:
			helper = "LlgoReadBool"
		case t.Info()&types.IsString != 0:
			helper = "LlgoReadString"
		case t.Info()&types.IsUnsigned != 0:
			helper = "LlgoReadUint"
		}
		fmt.Fprintf(b, "return %s.%s(d,v)\n", e.jsonV2, helper)
	case *types.Pointer:
		fmt.Fprintf(b, `
	if d.PeekKind() == 'n' {
		if _, err := d.ReadToken(); err != nil { return err }
		*v = nil
		return nil
	}
	if *v == nil { *v = new(%s) }
	return %s(d, *v)
`, e.alias(s.elem.typ), element)
	case *types.Array:
		e.openDecodeContainer('[', "")
		fmt.Fprintf(b, `
	i := 0
	for ; d.PeekKind() != ']'; i++ {
		if i >= len(*v) {
			if err := d.SkipValue(); err != nil { return err }
			continue
		}
		err := %s(d, &(*v)[i])
`, element)
		e.keepDecodeError()
		fmt.Fprintf(b, `
	}
	var zero %s
	for ; i < len(*v); i++ { (*v)[i] = zero }
`, e.alias(s.elem.typ))
		e.finishDecodeContainer()
	case *types.Slice:
		e.openDecodeContainer('[', "*v = nil")
		fmt.Fprintf(b, `
	*v = (*v)[:cap(*v)]
	i := 0
	for ; d.PeekKind() != ']'; i++ {
		if i == len(*v) {
			var zero %s
			*v = append(*v, zero)
			*v = (*v)[:cap(*v)]
		}
		err := %s(d, &(*v)[i])
		if err != nil && !%s.LlgoRecoverable(err) {
			*v = (*v)[:i+1]
			return err
		}
		if first == nil { first = err }
	}
	if i == 0 { *v = make(%s, 0) } else { *v = (*v)[:i] }
`, e.alias(s.elem.typ), element, e.jsonV2, typ)
		e.finishDecodeContainer()
	case *types.Map:
		e.openDecodeContainer('{', "*v = nil")
		fmt.Fprintf(b, `
	if *v == nil { *v = make(%s) }
	for d.PeekKind() != '}' {
		key, err := d.ReadToken()
		if err != nil { return err }
		name := key.String()
		var item %s
		err = %s(d, &item)
		(*v)[name] = item
`, typ, e.alias(s.elem.typ), element)
		e.keepDecodeError()
		fmt.Fprintln(b, "}")
		e.finishDecodeContainer()
	case *types.Struct:
		e.openDecodeContainer('{', "")
		fmt.Fprint(b, `
	for d.PeekKind() != '}' {
		key, err := d.ReadToken()
		if err != nil { return err }
		field := key.String()
		switch field {
`)
		for i, f := range s.fields {
			fmt.Fprintf(b, "case %q: err=%s(d,&v.%s)\n", f.name, fields[i], f.goName)
		}
		fmt.Fprintln(b, "default: switch {")
		for i, f := range s.fields {
			fmt.Fprintf(b, "case %s.EqualFold(field,%q): err=%s(d,&v.%s)\n", e.stringsPkg, f.name, fields[i], f.goName)
		}
		fmt.Fprintln(b, "default: if err=d.SkipValue();err!=nil {return err} } }")
		e.keepDecodeError()
		fmt.Fprintln(b, "}")
		e.finishDecodeContainer()
	}
	fmt.Fprintln(b, "}")
	return name
}

func (e *emitter) openDecodeContainer(kind rune, onNull string) {
	fmt.Fprintf(&e.body, `
	tok, err := d.ReadToken()
	if err != nil { return err }
	switch tok.Kind() {
	case 'n':
		%s
		return nil
	case %q:
	default:
		return %s.LlgoBadToken(d, v)
	}
	var first error
`, onNull, kind, e.jsonV2)
}

func (e *emitter) keepDecodeError() {
	fmt.Fprintf(&e.body, `
		if err != nil {
			if !%s.LlgoRecoverable(err) { return err }
			if first == nil { first = err }
		}
`, e.jsonV2)
}

func (e *emitter) finishDecodeContainer() {
	fmt.Fprint(&e.body, `
	if _, err := d.ReadToken(); err != nil { return err }
	return first
`)
}
