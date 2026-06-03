//go:build go1.26
// +build go1.26

package test

import "testing"

func TestBuiltinPrintGo126FloatFormat(t *testing.T) {
	got := runBuiltinPrintProbe(t)
	want := "" +
		"1e+07\n" +
		"(1e+07-1e+07i)\n" +
		"(1.5+0i)\n" +
		"NaN\n" +
		"+Inf\n" +
		"-Inf\n" +
		"(1+NaNi)\n" +
		"(1+Infi)\n" +
		"(1-Infi)\n"
	if got != want {
		t.Fatalf("builtin print output mismatch:\n got %q\nwant %q", got, want)
	}
}
