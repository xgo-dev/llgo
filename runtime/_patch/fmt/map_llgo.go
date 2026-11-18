//go:build llgo && wasip1 && wasm && go1.21

package fmt

import (
	"slices"
	"strconv"
)

func llgoSprintfStringIntMap(_ string, args ...any) string {
	m := args[0].(map[string]int)
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	buf := []byte("map[")
	for i, key := range keys {
		if i != 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, key...)
		buf = append(buf, ':')
		buf = strconv.AppendInt(buf, int64(m[key]), 10)
	}
	return string(append(buf, ']'))
}
