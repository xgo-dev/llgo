//go:build llgo && llgo.size.binary

package binary

//llgo:template llgoSize
func Size(v any) int

func llgoSize[T any](v T) int { return Size(v) }
