//go:build !go1.26
// +build !go1.26

package test

import "testing"

func TestBuiltinPrintLegacyFloatFormat(t *testing.T) {
	got := runBuiltinPrintProbe(t)
	wantGC := "" +
		"+1.000000e+007\n" +
		"(+1.000000e+007-1.000000e+007i)\n" +
		"(+1.500000e+000+0.000000e+000i)\n" +
		"NaN\n" +
		"+Inf\n" +
		"-Inf\n" +
		"(+1.000000e+000NaNi)\n" +
		"(+1.000000e+000+Infi)\n" +
		"(+1.000000e+000-Infi)\n"
	wantLLGo := "" +
		"+1.000000e+07\n" +
		"(+1.000000e+07-1.000000e+07i)\n" +
		"(+1.500000e+00+0.000000e+00i)\n" +
		"NaN\n" +
		"+Inf\n" +
		"-Inf\n" +
		"(+1.000000e+00NaNi)\n" +
		"(+1.000000e+00+Infi)\n" +
		"(+1.000000e+00-Infi)\n"
	if got != wantGC && got != wantLLGo {
		t.Fatalf("builtin print output mismatch:\n got %q\nwant one of:\n  %q\n  %q", got, wantGC, wantLLGo)
	}
}
