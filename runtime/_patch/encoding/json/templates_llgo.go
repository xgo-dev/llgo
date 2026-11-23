//go:build llgo

package json

//llgointernal:template:bind Marshal any helper=llgoMarshal body=llgoMarshalBody exclude-import=encoding/json/v2
//llgointernal:template:bind Unmarshal any helper=llgoUnmarshal body=llgoUnmarshalBody exclude-import=encoding/json/v2
//llgointernal:template:bind MarshalIndent any fallback
//llgointernal:template:bind NewEncoder any fallback
//llgointernal:template:bind NewDecoder any fallback

func llgoMarshal[T any](v T) ([]byte, error)       { return Marshal(v) }
func llgoUnmarshal[T any](data []byte, v *T) error { return Unmarshal(data, v) }

const llgoMarshalBody = llgoJSONShape + llgoJSONEncode + `
{{- .RequireBuiltins "append" "byte" "error" "int64" "len" "make" "nil" "string" "uint64" -}}
{{- if not (and (.HasImport "strconv") (.HasImport "unicode/utf8") (or (.HasImport "sort") (.HasImport "slices"))) -}}
  {{- reject -}}
{{- end }}
return {{ .Emit "encode" (.Arg 0) }}(nil, a0), nil
`

const llgoUnmarshalBody = llgoJSONShape + llgoJSONDecode + `
{{- .RequireBuiltins "append" "byte" "cap" "error" "len" "make" "new" "nil" -}}
{{- if not (.HasObject "encoding/json" "LlgoDecodeTyped") }}{{ reject }}{{ end }}
return {{ .Import "encoding/json" }}.LlgoDecodeTyped(a0, a1, {{ .Emit "decode" (elem (.Arg 1)) }})
`

// Each helper validates its own node. The compiler shares identical typed
// helpers and declines recursive expansion, leaving the standard codec intact.
const llgoJSONShape = `
{{- define "check" -}}
  {{- $t := . -}}
  {{- $k := kind $t -}}
  {{- if or (namedIn $t "encoding/json" "Number") (hasMethod $t "MarshalJSON" "MarshalText" "MarshalJSONTo") }}{{ reject }}{{ end -}}
  {{- if eq $k "slice" -}}
    {{- if eq (kind (elem $t)) "uint8" }}{{ reject }}{{ end -}}
  {{- else if eq $k "map" -}}
    {{- if not (isString (key $t)) }}{{ reject }}{{ end -}}
  {{- else if not (or (isInteger $t) (eq $k "bool" "string" "pointer" "array" "struct")) -}}
    {{- reject -}}
  {{- end -}}
{{- end -}}

{{- define "fields" -}}
  {{- $g := index . 0 -}}
  {{- $out := index . 1 -}}
  {{- $seen := dict -}}
  {{- range $i, $field := $g.Fields $g.Type -}}
    {{- $tag := $field.Tag.Get "json" -}}
    {{- if eq $tag "-" }}{{ continue }}{{ end -}}
    {{- if $field.Embedded }}{{ reject }}{{ end -}}
    {{- if not $field.Exported }}{{ continue }}{{ end -}}
    {{- $parts := split $tag "," 2 -}}
    {{- $name := index $parts 0 -}}
    {{- if not $name }}{{ $name = $field.Name }}{{ end -}}
    {{- if or (get $seen $name) (not (validRunes $name "!#$%&()*+-./:;<=>?@[]^_{|}~ ")) }}{{ reject }}{{ end -}}
    {{- $omit := false -}}
    {{- if eq (len $parts) 2 -}}
      {{- $opts := index $parts 1 -}}
      {{- if not (or (eq $opts "") (eq $opts "omitempty")) }}{{ reject }}{{ end -}}
      {{- $omit = eq $opts "omitempty" -}}
    {{- end -}}
    {{- set $seen $name true -}}
    {{- set $out (printf "%08d" $i) (dict "name" $name "field" $field "omit" $omit) -}}
  {{- end -}}
{{- end -}}
`

const llgoJSONEncode = `
{{- define "encode" -}}
  {{- $g := . -}}
  {{- $t := .Type -}}
  {{- $k := kind $t -}}
  {{- template "check" $t -}}
func {{ .Name }}(dst []byte, v {{ .TypeName $t }}) []byte {
  {{- if eq $k "bool" }}
  if v { return append(dst,"true"...) }
  return append(dst,"false"...)
  {{- else if eq $k "string" }}
  return {{ .Emit "quote" (stringType) }}(dst, string(v))
  {{- else if isUnsigned $t }}
  return {{ .Import "strconv" }}.AppendUint(dst, uint64(v), 10)
  {{- else if isInteger $t }}
  return {{ .Import "strconv" }}.AppendInt(dst, int64(v), 10)
  {{- else if eq $k "pointer" }}
  if v == nil { return append(dst,"null"...) }
  return {{ .Emit "encode" (elem $t) }}(dst, *v)
  {{- else if eq $k "slice" "array" }}
    {{- if eq $k "slice" }}
  if v == nil { return append(dst,"null"...) }
    {{- end }}
  dst = append(dst, '[')
  for i, x := range v {
    if i != 0 { dst = append(dst, ',') }
    dst = {{ .Emit "encode" (elem $t) }}(dst, x)
  }
  return append(dst, ']')
  {{- else if eq $k "map" }}
  if v == nil { return append(dst,"null"...) }
  keys := make([]string, 0, len(v))
  for k := range v { keys = append(keys, k) }
    {{- if .HasImport "sort" }}
  {{ .Import "sort" }}.Strings(keys)
    {{- else }}
  {{ .Import "slices" }}.Sort(keys)
    {{- end }}
  dst = append(dst, '{')
  for i, k := range keys {
    if i != 0 { dst = append(dst, ',') }
    dst = {{ .Emit "quote" (stringType) }}(dst, k)
    dst = append(dst, ':')
    dst = {{ .Emit "encode" (elem $t) }}(dst, v[k])
  }
  return append(dst, '}')
  {{- else if eq $k "struct" -}}
    {{- $fields := dict -}}
    {{- template "fields" (list . $fields) }}
  dst = append(dst, '{')
    {{- if $fields }}
  start := len(dst)
      {{- range $fields -}}
        {{- $v := printf "v.%s" .field.Name -}}
        {{- $name := replace (replace (replace .name "&" "\\u0026") "<" "\\u003c") ">" "\\u003e" -}}
        {{- if .omit }}
  if {{ template "nonempty" (list (kind .field.Type) $v) }} {
        {{- end }}
    if len(dst) != start { dst = append(dst, ',') }
    dst = append(dst, {{ printf "%q" (printf "\"%s\":" $name) }}...)
    dst = {{ $g.Emit "encode" .field.Type }}(dst, {{ $v }})
        {{- if .omit }}
  }
        {{- end -}}
      {{- end -}}
    {{- end }}
  return append(dst, '}')
  {{- end }}
}
{{- end -}}

{{- define "nonempty" -}}
  {{- $k := index . 0 -}}
  {{- $v := index . 1 -}}
  {{- if eq $k "array" "slice" "map" -}}
    len({{ $v }}) != 0
  {{- else if eq $k "pointer" -}}
    {{ $v }} != nil
  {{- else if eq $k "bool" -}}
    {{ $v }}
  {{- else if eq $k "string" -}}
    {{ $v }} != ""
  {{- else if eq $k "struct" -}}
    true
  {{- else -}}
    {{ $v }} != 0
  {{- end -}}
{{- end -}}

{{- define "quote" -}}
  {{- $utf8 := .Import "unicode/utf8" -}}
  {{- $replacement := "\\ufffd" -}}
  {{- if .HasImport "encoding/json/v2" }}{{ $replacement = "\uFFFD" }}{{ end -}}
func {{ .Name }}(dst []byte, s string) []byte {
	const hex = "0123456789abcdef"
	dst = append(dst, '"')
	for i := 0; i < len(s); {
		b := s[i]
		if b < {{ $utf8 }}.RuneSelf {
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
		r, n := {{ $utf8 }}.DecodeRuneInString(s[i:])
		switch {
		case r == {{ $utf8 }}.RuneError && n == 1:
			dst = append(dst, {{ printf "%q" $replacement }}...)
		case r == '\u2028' || r == '\u2029':
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[r&15])
		default:
			dst = append(dst, s[i:i+n]...)
		}
		i += n
	}
	return append(dst, '"')
}
{{- end -}}
`

const llgoJSONDecode = `
{{- define "decode" -}}
  {{- $g := . -}}
  {{- $t := .Type -}}
  {{- $k := kind $t -}}
  {{- template "check" $t -}}
  {{- if hasMethod $t "UnmarshalJSON" "UnmarshalJSONFrom" "UnmarshalText" }}{{ reject }}{{ end -}}
  {{- $typ := .TypeName $t -}}
  {{- $v2 := .Import "encoding/json/v2" -}}
func {{ .Name }}(d *{{ .Import "encoding/json/jsontext" }}.Decoder, v *{{ $typ }}) error {
  {{- if eq $k "pointer" -}}
    {{- $item := elem $t }}
  if d.PeekKind() == 'n' {
    if _, err := d.ReadToken(); err != nil { return err }
    *v = nil
    return nil
  }
  if *v == nil { *v = new({{ .TypeName $item }}) }
  return {{ .Emit "decode" $item }}(d, *v)
  {{- else if eq $k "array" "slice" "map" "struct" -}}
    {{- template "open" (list $v2 $k) }}
    {{- if eq $k "array" "slice" -}}
      {{- $item := elem $t -}}
      {{- if eq $k "slice" }}
  *v = (*v)[:cap(*v)]
      {{- end }}
  i := 0
  for ; d.PeekKind() != ']'; i++ {
      {{- if eq $k "array" }}
    if i >= len(*v) {
      if err := d.SkipValue(); err != nil { return err }
      continue
    }
      {{- else }}
    if i == len(*v) {
      var zero {{ .TypeName $item }}
      *v = append(*v, zero)
      *v = (*v)[:cap(*v)]
    }
      {{- end }}
    err := {{ .Emit "decode" $item }}(d, &(*v)[i])
      {{- template "keepError" (list $v2 $k) }}
  }
      {{- if eq $k "array" }}
  var zero {{ .TypeName $item }}
  for ; i < len(*v); i++ { (*v)[i] = zero }
      {{- else }}
  if i == 0 { *v = make({{ $typ }}, 0) } else { *v = (*v)[:i] }
      {{- end -}}
    {{- else if eq $k "map" -}}
      {{- $item := elem $t }}
  if *v == nil { *v = make({{ $typ }}) }
  for d.PeekKind() != '}' {
    key, err := d.ReadToken()
    if err != nil { return err }
    name := key.String()
    var item {{ .TypeName $item }}
    err = {{ .Emit "decode" $item }}(d, &item)
    (*v)[name] = item
      {{- template "keepError" (list $v2 $k) }}
  }
    {{- else -}}
      {{- $fields := dict -}}
      {{- template "fields" (list . $fields) }}
  for d.PeekKind() != '}' {
    key, err := d.ReadToken()
    if err != nil { return err }
    field := key.String()
    switch field {
      {{- range $fields }}
    case {{ printf "%q" .name }}: err = {{ $g.Emit "decode" .field.Type }}(d, &v.{{ .field.Name }})
      {{- end }}
    default:
      switch {
      {{- range $fields }}
      case {{ $g.Import "strings" }}.EqualFold(field, {{ printf "%q" .name }}):
        err = {{ $g.Emit "decode" .field.Type }}(d, &v.{{ .field.Name }})
      {{- end }}
      default: if err = d.SkipValue(); err != nil { return err }
      }
    }
      {{- template "keepError" (list $v2 $k) }}
  }
    {{- end }}
  if _, err := d.ReadToken(); err != nil { return err }
  return first
  {{- else -}}
    {{- $read := "Int" -}}
    {{- if eq $k "bool" }}{{ $read = "Bool" -}}
    {{- else if eq $k "string" }}{{ $read = "String" -}}
    {{- else if isUnsigned $t }}{{ $read = "Uint" -}}
    {{- end }}
  return {{ $v2 }}.LlgoRead{{ $read }}(d, v)
  {{- end }}
}
{{- end -}}

{{- define "open" -}}
  {{- $v2 := index . 0 -}}
  {{- $kind := index . 1 }}
  tok, err := d.ReadToken()
  if err != nil { return err }
  switch tok.Kind() {
  case 'n':
    {{- if eq $kind "map" "slice" }}
    *v = nil
    {{- end }}
    return nil
  case {{ if eq $kind "array" "slice" }}'['{{ else }}'{'{{ end }}:
  default:
    return {{ $v2 }}.LlgoBadToken(d, v)
  }
  var first error
{{- end -}}

{{- define "keepError" -}}
  {{- $v2 := index . 0 -}}
  {{- $kind := index . 1 }}
    if err != nil {
      if !{{ $v2 }}.LlgoRecoverable(err) {
        {{- if eq $kind "slice" }}
        *v = (*v)[:i+1]
        {{- end }}
        return err
      }
      if first == nil { first = err }
    }
{{- end -}}
`
