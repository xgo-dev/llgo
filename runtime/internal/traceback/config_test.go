package traceback

import "testing"

func TestParse(t *testing.T) {
	negative := int(-1)
	wantNegative := All
	if uint64(^uint(0)) == 0xffffffff {
		wantNegative |= uint64(uint32(negative) << Shift)
	}
	for _, tc := range []struct {
		text string
		want uint64
	}{
		{"none", 0}, {"single", 4}, {"", 4}, {"all", 6},
		{"system", 10}, {"crash", 11}, {"wer", 2},
		{"0", 2}, {"1", 6}, {"2", 10}, {"3", 14}, {"+2", 10},
		{"-0", 2}, {"-1", wantNegative}, {"+", 2}, {"- ", 2}, {"bad", 2},
		{"-2147483648", 2}, {"-2147483649", 2},
		{" 1", 2}, {"1 ", 2}, {"4294967296", 2},
		{"18446744073709551616", 2},
	} {
		if got := Parse(tc.text, false); got != tc.want {
			t.Errorf("Parse(%q) = %d, want %d", tc.text, got, tc.want)
		}
	}
	if got := Parse("wer", true); got != 11|WER || Level(got) != 2 {
		t.Fatalf("Windows wer = %#x, level %d", got, Level(got))
	}
}

func TestCombine(t *testing.T) {
	for _, tc := range []struct {
		name, environment, setting string
		windows                    bool
		want                       uint64
	}{
		{"higher setting", "single", "system", false, 10},
		{"higher environment", "system", "single", false, 10},
		{"independent flags", "all", "crash", false, 11},
		{"numeric levels", "1", "2", false, 10},
		{"windows WER floor", "wer", "all", true, 11 | WER},
		{"windows WER setting", "single", "wer", true, 11 | WER},
	} {
		t.Run(tc.name, func(t *testing.T) {
			environment := Parse(tc.environment, tc.windows)
			setting := Parse(tc.setting, tc.windows)
			if got := Combine(environment, setting); got != tc.want {
				t.Fatalf("Combine(%#x, %#x) = %#x, want %#x", environment, setting, got, tc.want)
			}
		})
	}
}
