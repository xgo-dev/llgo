//go:build llgo

package fmt

import "os"

// The compiler selects these entries only for constant formats containing
// literal text and escaped percent signs, with no formatting arguments.
// Keep Write (including empty writes), its count/error and the current Stdout.
func llgoPrintfLiteral(text string) (int, error) {
	return os.Stdout.Write([]byte(text))
}

func llgoSprintfLiteral(text string) string {
	return text
}
