//go:build llgo && wasip1 && llgo.size.wasi

package os

import "internal/poll"

// A closed WASI link may use this wrapper when no code observes the standard
// File globals. WASI has no LLGo poll registration for these descriptors.
// Keeping File.WriteString preserves write serialization and error handling.
var llgoClosedStdout = File{&file{
	pfd:         poll.FD{Sysfd: 1, IsStream: true, ZeroReadIsEOF: true},
	name:        "/dev/stdout",
	stdoutOrErr: true,
}}

func llgoWriteClosedStdout(s string) (int, error) {
	return llgoClosedStdout.WriteString(s)
}
