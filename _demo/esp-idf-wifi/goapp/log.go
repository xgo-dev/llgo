package main

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

func hex8(v uint32) string {
	const digits = "0123456789abcdef"
	var s [8]byte
	for i := 7; i >= 0; i-- {
		s[i] = digits[v&15]
		v >>= 4
	}
	return string(s[:])
}

// dec32 formats v in decimal (negatives with a leading minus). println has
// no number form without a separating space, and the udp probe's golden
// lines pin the exact "n=128" shape, so the probe concatenates this in.
func dec32(v int) string {
	neg := v < 0
	x := uint32(v)
	if neg {
		x = -uint32(v)
	}
	if x == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// errCode extracts the esp pipeline code an error carries, 0 for nil - the
// same number the pre-error API printed ("err = 0").
func errCode(err error) int32 {
	if err == nil {
		return 0
	}
	if e, ok := err.(*esperr.Error); ok {
		return e.Code
	}
	return -1
}

// errText formats an esp error as "<decimal code> <name>", exactly the two
// tokens the pre-error probes printed side by side via int(e) and
// tls.ErrName(e) ("32258 TLS_ERR_CERT_VERIFY"), so the acceptance-golden
// diagnostic lines stay byte-identical. Errors without a name (the
// peripheral esp_err wrappers) render as the bare code; a non-esp error
// degrades to its own Error() text.
func errText(err error) string {
	if e, ok := err.(*esperr.Error); ok {
		if e.Name != "" {
			return dec32(int(e.Code)) + " " + e.Name
		}
		return dec32(int(e.Code))
	}
	if err == nil {
		return "0"
	}
	return err.Error()
}
