//go:build llgo && !go1.21

package json

func llgoMarshalString(v string) ([]byte, error) {
	var e encodeState
	e.string(v, true)
	return e.Bytes(), nil
}
