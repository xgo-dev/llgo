//go:build llgo

package json

//llgo:template llgoMarshal
func Marshal(v any) ([]byte, error)

//llgo:template llgoUnmarshal
func Unmarshal(data []byte, v any) error

func llgoMarshal[T any](v T) ([]byte, error)       { return Marshal(v) }
func llgoUnmarshal[T any](data []byte, v *T) error { return Unmarshal(data, v) }
