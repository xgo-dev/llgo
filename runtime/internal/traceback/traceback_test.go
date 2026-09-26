package traceback

import (
	"strconv"
	"strings"
	"testing"
)

func TestTracebackWindow(t *testing.T) {
	for _, count := range []int{0, 1, 49, 50, 51, 64, 99, 100, 101, 150, 151, 5000} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var window Window
			var out []byte
			for i := 0; i < count; i++ {
				out = window.Append(out, Frame{
					Function: "frame" + strconv.Itoa(i),
					File:     "stack.go",
					Line:     i + 1,
					PC:       0x110,
					Entry:    0x100,
				})
			}
			got := string(window.Finish(out))
			var want strings.Builder
			for i := 0; i < count; i++ {
				if count > 100 && i == 50 {
					want.WriteString("..." + strconv.Itoa(count-100) + " frames elided...\n")
					i = count - 50
				}
				want.WriteString("frame" + strconv.Itoa(i) + "(...)\n\tstack.go:" + strconv.Itoa(i+1) + " +0x10\n")
			}
			if got != want.String() {
				t.Fatalf("traceback mismatch:\ngot:\n%s\nwant:\n%s", got, want.String())
			}
		})
	}
}

func TestTracebackFrameUnknown(t *testing.T) {
	got := string(AppendFrame(nil, Frame{PC: 0x1234}))
	if want := "pc=0x1234(...)\n\t???:0\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	got = string(AppendFrame(nil, Frame{Function: "test", PC: 0x100, Entry: 0x110}))
	if strings.Contains(got, "+0x") {
		t.Fatalf("negative PC offset: %s", got)
	}
}

func TestAppendNumbers(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, v := range []int{0, 1, -1, 100, -100, maxInt, -maxInt - 1} {
		if got, want := string(AppendInt(nil, v)), strconv.Itoa(v); got != want {
			t.Errorf("AppendInt(%d) = %q, want %q", v, got, want)
		}
	}
	for _, v := range []uintptr{0, 1, 0xff, 0x100, ^uintptr(0)} {
		if got, want := string(AppendHex(nil, v)), strconv.FormatUint(uint64(v), 16); got != want {
			t.Errorf("AppendHex(%x) = %q, want %q", v, got, want)
		}
	}
}
