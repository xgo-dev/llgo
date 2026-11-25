package identity

import "runtime"

const Name = runtime.Compiler
const Description = Name + " toolchain"

type NameBytes [len(Name)]byte

func Branch() string {
	switch runtime.Compiler {
	case "gc":
		return "gc-branch"
	case "llgo":
		return "llgo-branch"
	default:
		return "other-branch"
	}
}
