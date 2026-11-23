package abi

// NilReceiverPanicMessage formats the panic for a value method on a nil pointer.
// The compiler can precompute this message when both names are known.
func NilReceiverPanicMessage(receiver, method string) string {
	const commandLinePrefix = "command-line-arguments."
	if len(receiver) > len(commandLinePrefix) && receiver[:len(commandLinePrefix)] == commandLinePrefix {
		receiver = "main." + receiver[len(commandLinePrefix):]
	}
	return "value method " + receiver + "." + method + " called using nil *" + unqualifiedReceiver(receiver) + " pointer"
}

func unqualifiedReceiver(receiver string) string {
	depth := 0
	for i := len(receiver) - 1; i >= 0; i-- {
		switch receiver[i] {
		case ']':
			depth++
		case '[':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				return receiver[i+1:]
			}
		}
	}
	return receiver
}
