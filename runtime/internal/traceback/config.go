package traceback

import "unsafe"

// The low bits have the same meaning as Go's runtime/runtime1.go.
const (
	Crash uint64 = 1 << iota
	All
	Shift = iota
	// WER is private to LLGo; it preserves Windows Error Reporting on crash.
	WER uint64 = 1 << 32
)

func Parse(level string, windows bool) uint64 {
	switch level {
	case "none":
		return 0
	case "", "single":
		return 1 << Shift
	case "all":
		return 1<<Shift | All
	case "system":
		return 2<<Shift | All
	case "crash":
		return 2<<Shift | All | Crash
	case "wer":
		if windows {
			return 2<<Shift | All | Crash | WER
		}
	}
	// Go retains the historical numeric forms and suppresses traces for
	// invalid values. Match Atoi's sign and native-int overflow checks too.
	s := level
	negative := false
	start := 0
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		negative = s[0] == '-'
		start = 1
	}
	if len(s) == start {
		return All
	}
	var n uint64
	limit := uint64(^uint(0) >> 1)
	if negative {
		limit++ // Atoi accepts the native minimum int.
	}
	if limit > 0xffffffff {
		limit = 0xffffffff
	}
	data := unsafe.StringData(s)
	if data == nil {
		return All
	}
	// The loop bounds guard the read. Startup can parse this setting
	// before ordinary panic handling is available.
	for i := start; i < len(s); i++ {
		ch := *(*byte)(unsafe.Add(unsafe.Pointer(data), i))
		if ch < '0' || ch > '9' {
			return All
		}
		d := uint64(ch - '0')
		if n > (limit-d)/10 {
			return All
		}
		n = n*10 + d
	}
	if negative {
		// On 32-bit Go, int(uint32(n)) round-trips negative ints too.
		if uint64(^uint(0)) > 0xffffffff && n != 0 {
			return All
		}
		n = -n
	}
	return All | uint64(uint32(n)<<Shift)
}

func Level(setting uint64) uint32 { return uint32(setting) >> Shift }

// Combine keeps the higher traceback detail level while preserving independent
// flags from both the environment and the program setting.
func Combine(environment, setting uint64) uint64 {
	level := Level(environment)
	if requested := Level(setting); requested > level {
		level = requested
	}
	return uint64(level)<<Shift | (environment|setting)&(Crash|All|WER)
}
