// Package traceback formats bounded Go-style stack traces without depending on
// the scheduler or platform unwinder.
package traceback

const (
	// Match runtime/traceback.go in Go: a traceback contains at most the
	// innermost 50 and outermost 50 logical frames.
	InnerFrames = 50
	OuterFrames = 50

	// Guard physical walks against corrupt but apparently monotonic chains.
	// This is not the display limit, nor the size of a Callers output buffer.
	MaxFrames = 1 << 20
)

// Frame contains the information needed to format a logical call frame.
type Frame struct {
	PC, Entry      uintptr
	Function, File string
	Line           int
}

type Window struct {
	count int
	outer [OuterFrames]Frame
}

func (w *Window) Append(out []byte, frame Frame) []byte {
	if w.count < InnerFrames {
		out = AppendFrame(out, frame)
	} else {
		w.outer[(w.count-InnerFrames)%len(w.outer)] = frame
	}
	w.count++
	return out
}

func (w *Window) Finish(out []byte) []byte {
	if w.count > InnerFrames+OuterFrames {
		out = append(out, "..."...)
		out = AppendInt(out, w.count-InnerFrames-OuterFrames)
		out = append(out, " frames elided...\n"...)
	}
	count := w.count - InnerFrames
	start := 0
	if count > len(w.outer) {
		start = count % len(w.outer)
		count = len(w.outer)
	}
	for i := 0; i < count; i++ {
		out = AppendFrame(out, w.outer[(start+i)%len(w.outer)])
	}
	return out
}

func AppendFrame(out []byte, frame Frame) []byte {
	name := frame.Function
	if name == "" {
		name = "pc=0x" + string(AppendHex(nil, frame.PC))
	}
	out = append(out, name...)
	out = append(out, "(...)\n\t"...)
	if frame.File == "" {
		out = append(out, "???"...)
	} else {
		out = append(out, frame.File...)
	}
	out = append(out, ':')
	out = AppendInt(out, frame.Line)
	if frame.Entry != 0 && frame.PC >= frame.Entry {
		out = append(out, " +0x"...)
		out = AppendHex(out, frame.PC-frame.Entry)
	}
	return append(out, '\n')
}

func AppendHex(buf []byte, v uintptr) []byte {
	const digits = "0123456789abcdef"
	if v == 0 {
		return append(buf, '0')
	}
	var tmp [16]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = digits[v&0xf]
		v >>= 4
	}
	return append(buf, tmp[i:]...)
}

func AppendInt(out []byte, v int) []byte {
	if v == 0 {
		return append(out, '0')
	}
	n := uint(v)
	if v < 0 {
		out = append(out, '-')
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return append(out, digits[i:]...)
}
