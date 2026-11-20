//go:build llgo && ((wasip1 && !llgo.wasm.debugger) || llgo.size.native || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded))

package fmt

import "os"

//llgo:skip Printf Print Println

// llgoStdoutWriter shares os.File's representation but exposes only the
// operation the fmt writer entry points use. Boxing *os.File here roots its
// full method set, including unrelated file, poll and time operations.
type llgoStdoutWriter os.File

func (f *llgoStdoutWriter) Write(p []byte) (int, error) {
	return (*os.File)(f).Write(p)
}

// Printf formats its arguments and writes to the current standard output.
func Printf(format string, a ...any) (n int, err error) {
	return Fprintf((*llgoStdoutWriter)(os.Stdout), format, a...)
}

// Print formats its arguments and writes to the current standard output.
func Print(a ...any) (n int, err error) {
	return Fprint((*llgoStdoutWriter)(os.Stdout), a...)
}

// Println formats its arguments with spaces and a trailing newline.
func Println(a ...any) (n int, err error) {
	return Fprintln((*llgoStdoutWriter)(os.Stdout), a...)
}
