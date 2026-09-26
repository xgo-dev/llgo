//go:build baremetal

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/traceback"

const tracebackLLGoFiles = ""

var tracebackSetting uint64 = 1 << traceback.Shift

func TracebackSetting() uint64 { return tracebackSetting }

// Baremetal has no GOTRACEBACK environment floor to preserve.
func SetTraceback(level string) { tracebackSetting = traceback.Parse(level, false) }
func TracebackEnabled() bool    { return traceback.Level(tracebackSetting) != 0 }
func crashAfterPanic()          {}
