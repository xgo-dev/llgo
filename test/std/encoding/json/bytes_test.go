package json_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

type byteKind uint8

func TestByteSlicesUseBase64(t *testing.T) {
	for _, input := range []any{[]byte{13}, []byteKind{13}} {
		data, err := json.Marshal(input)
		if err != nil || string(data) != `"DQ=="` {
			t.Fatalf("marshal %T: %s, %v; want base64", input, data, err)
		}
		var got []byteKind
		if err := json.Unmarshal(data, &got); err != nil || !reflect.DeepEqual(got, []byteKind{13}) {
			t.Fatalf("byte slice round trip: %v, %v", got, err)
		}
	}
}
