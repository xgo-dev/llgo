//go:build !baremetal

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

//go:linkname tracebackGetenv C.getenv
func tracebackGetenv(*c.Char) *c.Char

//go:linkname tracebackCrash C.llgo_traceback_crash
func tracebackCrash()

//go:linkname tracebackPlatformConfig C.llgo_traceback_config
func tracebackPlatformConfig(initialize, wer bool)

const tracebackLLGoFiles = "; _wrap/traceback_crash.c"

var tracebackSetting uint64 = 1 << traceback.Shift
var tracebackEnvironment uint64

func init() {
	level := ""
	if env := tracebackGetenv(c.Str("GOTRACEBACK")); env != nil {
		level = c.GoString(env)
	}
	tracebackEnvironment = traceback.Parse(level, tracebackWindows)
	tracebackPlatformConfig(true, tracebackEnvironment&traceback.WER != 0)
	atomic.Store(&tracebackSetting, tracebackEnvironment)
}

func TracebackSetting() uint64 { return atomic.Load(&tracebackSetting) }

func SetTraceback(level string) {
	// Like Go, a program may increase the environment-selected detail but
	// cannot reduce it. Changes to the environment after startup do not count.
	setting := traceback.Parse(level, tracebackWindows)
	tracebackPlatformConfig(false, setting&traceback.WER != 0)
	atomic.Store(&tracebackSetting, traceback.Combine(tracebackEnvironment, setting))
}

func crashAfterPanic() {
	setting := TracebackSetting()
	if setting&traceback.Crash != 0 {
		tracebackCrash()
	}
}

func TracebackEnabled() bool { return traceback.Level(TracebackSetting()) != 0 }
