package abi

import "testing"

func TestNilReceiverPanicMessage(t *testing.T) {
	for _, tt := range []struct {
		receiver string
		want     string
	}{
		{"T", "value method T.M called using nil *T pointer"},
		{"command-line-arguments.T", "value method main.T.M called using nil *T pointer"},
		{
			"example.com/pkg.T[example.com/other.U[example.com/args.V]]",
			"value method example.com/pkg.T[example.com/other.U[example.com/args.V]].M called using nil *T[example.com/other.U[example.com/args.V]] pointer",
		},
	} {
		if got := NilReceiverPanicMessage(tt.receiver, "M"); got != tt.want {
			t.Errorf("NilReceiverPanicMessage(%q, %q) = %q, want %q", tt.receiver, "M", got, tt.want)
		}
	}
}
