//go:build llgo && goexperiment.jsonv2

package json

import (
	"encoding/json/internal/jsonopts"
	"encoding/json/internal/jsonwire"
)

func llgoMarshalString(v string) ([]byte, error) {
	return jsonwire.AppendQuote(nil, []byte(v), &jsonopts.DefaultOptionsV1.Flags)
}
