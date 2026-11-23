//go:build llgo && llgo.size.binary && go1.23

package binary

//llgointernal:template:bind Encode any helper=llgoEncode body=llgoEncodeBody
//llgointernal:template:bind Decode any helper=llgoDecode body=llgoDecodeBody
//llgointernal:template:bind Append any fallback

func llgoEncode[O ByteOrder, T any](buf []byte, order O, v T) (int, error) {
	return Encode(buf, order, v)
}

func llgoDecode[O ByteOrder, T any](buf []byte, order O, v T) (int, error) {
	return Decode(buf, order, v)
}

// LLGoBinaryShortBuffer keeps the error object used by Encode and Decode.
func LLGoBinaryShortBuffer() error { return errBufferTooSmall }

// Reflect widens float32 before converting back to wire bits. Keep that
// boundary through LTO so signaling NaNs retain the same behavior.
//
//go:noinline
func LLGoBinaryWiden(v float32) float64 { return float64(v) }

const llgoEncodeBody = llgoTransferBody + `{{template "transfer" (list . "encode")}}`
const llgoDecodeBody = llgoTransferBody + `{{template "transfer" (list . "decode")}}`

const llgoTransferBody = llgoWireLayout + `
{{- define "transfer" -}}
  {{- $g := index . 0 -}}
  {{- $op := index . 1 -}}
  {{- $g.RequireBuiltins "bool" "byte" "complex" "error" "false" "float32" "float64" "imag" "int" "len" "nil" "real" "true" "uint16" "uint32" "uint64" -}}
  {{- if not (namedIn ($g.Arg 1) "encoding/binary" "bigEndian" "littleEndian" "nativeEndian") }}{{ reject }}{{ end -}}
  {{- $t := $g.Arg 2 -}}
  {{- $v := "a2" -}}
  {{- $ptr := eq (kind $t) "pointer" -}}
  {{- if $ptr -}}
    {{- if not ($g.NonNil 2) }}{{ reject }}{{ end -}}
    {{- $t = elem $t -}}
    {{- $v = "(*a2)" -}}
  {{- end -}}
  {{- $slice := eq (kind $t) "slice" -}}
  {{- if $slice }}{{ $t = elem $t }}{{ else if and (eq $op "decode") (not $ptr) }}{{ reject }}{{ end -}}
  {{- if not (or (eq (kind $t) "array") (eq (kind $t) "struct")) }}{{ reject }}{{ end -}}
  {{- $wire := dict -}}
  {{- template "layout" (list $g $t $wire (dict "remaining" (add 0 256)) $op) -}}
n := {{ if $slice }}len({{ $v }})*{{ end }}{{ $wire.width }}
  {{- if or $slice (and (ne $wire.width 0) (lt ($g.SliceLength 0) $wire.width)) }}
if len(a0) < n { return 0, {{ $g.Import "encoding/binary" }}.LLGoBinaryShortBuffer() }
  {{- end }}
  {{- if and $slice (ne $wire.width 0) }}
for i := range {{ $v }} {
  {{- template "field" (list $g $wire (printf "%s[i]" $v) (printf "(i*%d)" $wire.width) $op) }}
}
  {{- else if not $slice }}
    {{- template "field" (list $g $wire $v "0" $op) }}
  {{- end }}
return n, nil
{{- end -}}

{{- define "field" -}}
  {{- $g := index . 0 -}}
  {{- $wire := index . 1 -}}
  {{- $v := index . 2 -}}
  {{- $off := index . 3 -}}
  {{- $op := index . 4 -}}
  {{- if eq $wire.width 0 }}{{ else if eq $wire.kind "array" -}}
    {{- $i := $g.Local }}
for {{ $i }} := range {{ $v }} {
    {{- template "field" (list $g $wire.elem (printf "%s[%s]" $v $i) (printf "(%s+%s*%d)" $off $i $wire.elem.width) $op) }}
}
  {{- else if eq $wire.kind "struct" -}}
    {{- range $wire.fields -}}
      {{- $next := $off -}}
      {{- if ne .offset 0 }}{{ $next = printf "(%s+%d)" $off .offset }}{{ end -}}
      {{- if ne .name "_" -}}
        {{- template "field" (list $g . (printf "%s.%s" $v .name) $next $op) -}}
      {{- else if and (eq $op "encode") (ne .width 0) -}}
        {{- $i := $g.Local }}
for {{ $i }} := {{ $next }}; {{ $i }} < ({{ $next }}+{{ .width }}); {{ $i }}++ { a0[{{ $i }}] = 0 }
      {{- end -}}
    {{- end -}}
  {{- else if eq $wire.kind "bool" }}
    {{- if eq $op "decode" }}
{{ $v }} = {{ $g.TypeName $wire.type }}(a0[{{ $off }}] != 0)
    {{- else }}
if {{ $v }} { a0[{{ $off }}] = 1 } else { a0[{{ $off }}] = 0 }
    {{- end -}}
  {{- else if eq $wire.kind "complex64" "complex128" -}}
    {{- $width := div $wire.width 2 -}}
    {{- $second := printf "(%s+%d)" $off $width -}}
    {{- if eq $op "decode" }}
{{ $v }} = {{ $g.TypeName $wire.type }}(complex({{template "read" (list $g $width $off true)}}, {{template "read" (list $g $width $second true)}}))
    {{- else -}}
      {{- $tmp := $g.Local }}
{{ $tmp }} := {{ $v }}
      {{- template "write" (list $g $width $off (printf "real(%s)" $tmp) true) }}
      {{- template "write" (list $g $width $second (printf "imag(%s)" $tmp) true) }}
    {{- end -}}
  {{- else -}}
    {{- $float := or (eq $wire.kind "float32") (eq $wire.kind "float64") -}}
    {{- if eq $op "decode" }}
{{ $v }} = {{ $g.TypeName $wire.type }}({{template "read" (list $g $wire.width $off $float)}})
    {{- else -}}
      {{- template "write" (list $g $wire.width $off $v $float) -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{- define "read" -}}
  {{- $g := index . 0 -}}
  {{- $width := index . 1 -}}
  {{- $off := index . 2 -}}
  {{- $float := index . 3 -}}
  {{- if eq $width 1 -}}
    a0[{{ $off }}]
  {{- else -}}
    {{- $bits := mul $width 8 -}}
    {{- if $float -}}
      {{- if eq $width 4 }}float32({{ $g.Import "encoding/binary" }}.LLGoBinaryWiden({{end -}}
      {{- $g.Import "math" }}.Float{{ $bits }}frombits(
    {{- end -}}
    a1.Uint{{ $bits }}(a0[{{ $off }}:({{ $off }}+{{ $width }})])
    {{- if $float }}{{ if eq $width 4 }})){{ end }}){{ end -}}
  {{- end -}}
{{- end -}}

{{- define "write" -}}
  {{- $g := index . 0 -}}
  {{- $width := index . 1 -}}
  {{- $off := index . 2 -}}
  {{- $v := index . 3 -}}
  {{- $float := index . 4 -}}
  {{- if eq $width 1 }}
a0[{{ $off }}] = byte({{ $v }})
  {{- else -}}
    {{- $bits := mul $width 8 }}
a1.PutUint{{ $bits }}(a0[{{ $off }}:({{ $off }}+{{ $width }})], uint{{ $bits }}(
    {{- if $float -}}
      {{- $g.Import "math" }}.Float{{ $bits }}bits(
      {{- if eq $width 4 }}float32({{ $g.Import "encoding/binary" }}.LLGoBinaryWiden({{ end -}}
      float{{ $bits }}({{ $v }})
      {{- if eq $width 4 }})){{ end -}})
    {{- else -}}
      {{- $v -}}
    {{- end -}}
))
  {{- end -}}
{{- end -}}
`
