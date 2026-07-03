//go:build !llgo
// +build !llgo

package ssa

import "testing"

func TestNeedsFramePointer(t *testing.T) {
	cases := []struct {
		target *Target
		want   bool
	}{
		{nil, true},
		{&Target{GOOS: "linux", GOARCH: "amd64"}, true},
		{&Target{GOOS: "darwin", GOARCH: "arm64"}, true},
		{&Target{GOOS: "linux", GOARCH: "wasm"}, false},
		{&Target{GOOS: "linux", GOARCH: "riscv32", Target: "esp32c3"}, false},
		{&Target{GOOS: "windows", GOARCH: "amd64"}, false},
	}
	for _, c := range cases {
		p := NewProgram(c.target)
		if got := p.NeedsFramePointer(); got != c.want {
			t.Fatalf("NeedsFramePointer(%+v) = %v, want %v", c.target, got, c.want)
		}
	}
}
