//go:build llgo && go1.21 && !goexperiment.jsonv2

package json

func llgoMarshalString(v string) ([]byte, error) {
	return appendString(nil, v, true), nil
}
