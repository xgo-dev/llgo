//go:build llgo

package fmt

import "os"

func llgoPrintfScalars(format string, args ...any) (int, error) {
	var storage [64]byte
	return os.Stdout.Write(llgoFormatScalars(storage[:0], format, args))
}

func llgoSprintfScalars(format string, args ...any) string {
	var storage [64]byte
	return string(llgoFormatScalars(storage[:0], format, args))
}

// LLGo selects these entry points only for validated constant formats with
// matching builtin scalar arguments. Use fmt's existing padding and digits.
func llgoFormatScalars(buf buffer, format string, args []any) buffer {
	var f fmt
	f.init(&buf)
	start := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		buf.writeString(format[start:i])
		i++
		if format[i] == '%' {
			buf.writeByte('%')
			start = i + 1
			continue
		}
		f.clearflags()
		f.zero = format[i] == '0'
		if f.zero {
			i++
		}
		f.wid, f.widPresent, i = parsenum(format, i, len(format))
		value := args[0]
		args = args[1:]
		switch format[i] {
		case 's':
			f.fmtS(value.(string))
		case 't':
			f.fmtBoolean(value.(bool))
		default:
			u, signed := llgoFormatInteger(value)
			base, digits := 10, ldigits
			switch format[i] {
			case 'b':
				base = 2
			case 'o':
				base = 8
			case 'x':
				base = 16
			case 'X':
				base, digits = 16, udigits
			}
			f.fmtInteger(u, base, signed, rune(format[i]), digits)
		}
		start = i + 1
	}
	buf.writeString(format[start:])
	return buf
}

func llgoFormatInteger(value any) (uint64, bool) {
	switch v := value.(type) {
	case int:
		return uint64(v), true
	case int8:
		return uint64(v), true
	case int16:
		return uint64(v), true
	case int32:
		return uint64(v), true
	case int64:
		return uint64(v), true
	case uint:
		return uint64(v), false
	case uint8:
		return uint64(v), false
	case uint16:
		return uint64(v), false
	case uint32:
		return uint64(v), false
	case uint64:
		return v, false
	case uintptr:
		return uint64(v), false
	}
	panic("fmt: invalid scalar integer")
}
