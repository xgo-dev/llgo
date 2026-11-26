package main

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

func trimEOL(s string) string {
	// Keep the first line only, and drop its trailing CR/LF.
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			s = s[:i]
			break
		}
	}
	return s
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// tagl builds the prefix shared by every keyboard log line: "I (<tick>) kb:"
// - the same shape the ESP-IDF lines around it use, so the console reads as
// one ordered stream. The tick is the esp_timer millisecond clock, the
// domain the C-side logs timestamp in.
func tagl() string {
	return "I (" + dec32(nowMs()) + ") kb:"
}

func errText(err error) string {
	if err == nil {
		return "0"
	}
	if e, ok := err.(*esperr.Error); ok {
		if e.Name != "" {
			return dec32(uint32(e.Code)) + " " + e.Name
		}
		return dec32(uint32(e.Code))
	}
	return "nonzero"
}

// dec32 formats v in decimal (no strconv: the stdlib drags reflect
// machinery into a bare-metal image).
func dec32(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

const hexDigits = "0123456789abcdef"

func hexNib(v uint8) byte { return hexDigits[v&0xF] }

func hex8(v uint8) string {
	var b [2]byte
	b[0], b[1] = hexNib(v>>4), hexNib(v)
	return string(b[:])
}

func hex16(v uint16) string {
	var b [4]byte
	for i := 0; i < 4; i++ {
		b[3-i] = hexNib(uint8(v >> (uint(i) * 4)))
	}
	return string(b[:])
}

func joinStrings(parts []string, sep string) string {
	n := 0
	for _, s := range parts {
		n += len(s)
	}
	n += len(sep) * len(parts)
	out := make([]byte, 0, n)
	for i, s := range parts {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, s...)
	}
	return string(out)
}

func hexBytes(data []uint8) string {
	if len(data) == 0 {
		return "-"
	}
	out := make([]byte, 0, len(data)*3)
	for i, c := range data {
		if i > 0 {
			out = append(out, ' ')
		}
		out = append(out, hexDigits[c>>4], hexDigits[c&0xF])
	}
	return string(out)
}
