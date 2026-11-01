package abi

import (
	"strconv"
	"testing"
)

func TestPublicLayoutTypeString(t *testing.T) {
	metadata := ` llgo.layout:"v1;p=1"`
	user := `json:"x\\\"y"`
	for _, tc := range []struct{ input, want string }{
		{`struct{ X int }`, `struct{ X int }`},
		{`struct{ X int ` + strconv.Quote(metadata) + ` }`, `struct{ X int }`},
		{`struct{ X int ` + strconv.Quote(user+metadata) + `; Y byte ` + strconv.Quote(metadata) + ` }`, `struct{ X int ` + strconv.Quote(user) + `; Y byte }`},
		{`struct{ X int "llgo.layout:\`, `struct{ X int "llgo.layout:\`},
	} {
		if got := publicLayoutTypeString(tc.input); got != tc.want {
			t.Errorf("publicLayoutTypeString(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
