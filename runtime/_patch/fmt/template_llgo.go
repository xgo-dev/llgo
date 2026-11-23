//go:build llgo

package fmt

import "os"

// These entries receive already validated operations from LLGo. Build the
// complete buffer before writing so Printf retains the single-write contract.
func LLGoTemplateStdout(out []byte) (int, error) {
	return os.Stdout.Write(out)
}

func LLGoTemplateInteger(out []byte, value uint64, signed bool, verb rune, width int, zero bool) []byte {
	buf := buffer(out)
	var f fmt
	f.init(&buf)
	f.wid, f.widPresent, f.zero = width, width != 0, zero
	base, digits := 10, ldigits
	switch verb {
	case 'b':
		base = 2
	case 'o':
		base = 8
	case 'x':
		base = 16
	case 'X':
		base, digits = 16, udigits
	}
	f.fmtInteger(value, base, signed, verb, digits)
	return []byte(buf)
}

func LLGoTemplateString(out []byte, value string, verb rune, width int, zero bool) []byte {
	buf := buffer(out)
	var f fmt
	f.init(&buf)
	f.wid, f.widPresent, f.zero = width, width != 0, zero
	switch verb {
	case 'q':
		f.fmtQ(value)
	case 'x':
		f.fmtSx(value, ldigits)
	case 'X':
		f.fmtSx(value, udigits)
	default:
		f.fmtS(value)
	}
	return []byte(buf)
}

func LLGoTemplateBool(out []byte, value bool, _ rune, width int, zero bool) []byte {
	buf := buffer(out)
	var f fmt
	f.init(&buf)
	f.wid, f.widPresent, f.zero = width, width != 0, zero
	f.fmtBoolean(value)
	return []byte(buf)
}
