package quoted

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitMatchesGoCommandQuoting(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   string
		want    []string
		wantErr string
	}{
		{name: "empty", value: "", want: nil},
		{name: "space", value: " ", want: nil},
		{name: "two", value: "a  b", want: []string{"a", "b"}},
		{name: "single quote", value: `'a b'`, want: []string{"a b"}},
		{name: "double quote", value: `"a b"`, want: []string{"a b"}},
		{name: "adjacent quotes", value: `'a '"b "`, want: []string{"a ", "b "}},
		{name: "no unescaping", value: `\'`, want: []string{`\'`}},
		{name: "unterminated", value: `'a`, wantErr: "unterminated ' string"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Split(test.value)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("Split(%q) error = %v, want %q", test.value, err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Split(%q) = %#v, want %#v", test.value, got, test.want)
			}
		})
	}
}
