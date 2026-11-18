//go:build llgo && wasip1 && wasm && llgo.fmt.value

package fmt

import "internal/llgofmt"

func llgoSprintfValue(_ string, args ...any) string {
	return llgofmt.Value(args[0])
}

func llgoSprintfValueWithMaps(_ string, args ...any) string {
	return llgofmt.ValueWithMaps(args[0])
}
