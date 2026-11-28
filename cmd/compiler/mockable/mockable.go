package mockable

import (
	"os"
)

var (
	exitFunc = os.Exit
	exitCode int
)

// EnableMock enables mocking of os.Exit
func EnableMock() {
	exitCode = 0
	exitFunc = func(code int) {
		exitCode = code
		panic("exit")
	}
}

// DisableMock restores the default os.Exit behavior
func DisableMock() {
	exitFunc = os.Exit
}

// Exit calls the current exit function
func Exit(code int) {
	exitFunc(code)
}

// ExitCode returns the last exit code from a mocked Exit call
func ExitCode() int {
	return exitCode
}
