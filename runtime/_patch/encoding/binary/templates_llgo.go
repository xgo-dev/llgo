//go:build llgo && llgo.size.binary

package binary

//llgointernal:template:bind Size any helper=llgoSize body=llgoSizeBody
//llgointernal:template:bind Read any fallback
//llgointernal:template:bind Write any fallback

func llgoSize[T any](v T) int { return Size(v) }

const llgoSizeBody = llgoWireLayout + `
{{- .RequireBuiltins "int" "len" "nil" -}}
{{- $t := .Arg 0 -}}
{{- $ptr := eq (kind $t) "pointer" -}}
{{- if $ptr }}{{ $t = elem $t }}{{ end -}}
{{- $slice := eq (kind $t) "slice" -}}
{{- if $slice }}{{ $t = elem $t }}{{ end -}}
{{- if not (or (eq (kind $t) "array") (eq (kind $t) "struct")) }}{{ reject }}{{ end -}}
{{- $wire := dict -}}
{{- template "layout" (list . $t $wire (dict "remaining" (add 0 256)) "size") -}}
{{- if $ptr }}
if a0 == nil { return -1 }
{{- end }}
return {{if $slice}}len({{if $ptr}}*{{end}}a0)*{{end}}{{ $wire.width }}
`

// The layout uses wire widths, including explicit blank fields. The target's
// Go struct padding and alignment play no part in this calculation.
const llgoWireLayout = `
{{- define "layout" -}}
  {{- $g := index . 0 -}}
  {{- $t := index . 1 -}}
  {{- $out := index . 2 -}}
  {{- $budget := index . 3 -}}
  {{- $op := index . 4 -}}
  {{- if eq $budget.remaining 0 }}{{ reject }}{{ end -}}
  {{- set $budget "remaining" (sub $budget.remaining 1) -}}
  {{- $k := kind $t -}}
  {{- $width := add 0 0 -}}
  {{- if eq $k "bool" "int8" "uint8" -}}
    {{- $width = add 0 1 -}}
  {{- else if eq $k "int16" "uint16" -}}
    {{- $width = add 0 2 -}}
  {{- else if eq $k "int32" "uint32" "float32" -}}
    {{- $width = add 0 4 -}}
  {{- else if eq $k "int64" "uint64" "float64" "complex64" -}}
    {{- $width = add 0 8 -}}
  {{- else if eq $k "complex128" -}}
    {{- $width = add 0 16 -}}
  {{- else if eq $k "array" -}}
    {{- $item := dict -}}
    {{- template "layout" (list $g (elem $t) $item $budget $op) -}}
    {{- $n := arrayLength $t -}}
    {{- if and (ne $item.width 0) (gt $n (div $g.IntMax $item.width)) }}{{ reject }}{{ end -}}
    {{- $width = mul $n $item.width -}}
    {{- set $out "elem" $item -}}
  {{- else if eq $k "struct" -}}
    {{- $fields := dict -}}
    {{- range $i, $field := $g.Fields $t -}}
      {{- if and (ne $op "size") (ne $field.Name "_") (not $field.Exported) (or (eq $op "decode") (not $field.Accessible)) }}{{ reject }}{{ end -}}
      {{- $shape := dict "name" $field.Name "offset" $width -}}
      {{- $fieldOp := $op -}}
      {{- if eq $field.Name "_" }}{{ $fieldOp = "size" }}{{ end -}}
      {{- template "layout" (list $g $field.Type $shape $budget $fieldOp) -}}
      {{- if gt $shape.width (sub $g.IntMax $width) }}{{ reject }}{{ end -}}
      {{- $width = add $width $shape.width -}}
      {{- set $fields (printf "%08d" $i) $shape -}}
    {{- end -}}
    {{- set $out "fields" $fields -}}
  {{- else -}}
    {{- reject -}}
  {{- end -}}
  {{- if gt $width $g.IntMax }}{{ reject }}{{ end -}}
  {{- set $out "kind" $k -}}
  {{- set $out "type" $t -}}
  {{- set $out "width" $width -}}
{{- end -}}
`
