package traceback

// Visible matches Go's default frame selection, with LLGo's implementation
// packages treated as runtime internals. Panic boundaries remain visible when
// deferred user code appears above them.
func Visible(name string, system, first bool) bool {
	if system {
		return true
	}
	if name == "runtime.runFinalizers" || name == "runtime.runCleanups" {
		return true
	}
	if !first && (name == "runtime.gopanic" || name == "github.com/xgo-dev/llgo/runtime/internal/runtime.Panic") {
		return true
	}
	const core = "github.com/xgo-dev/llgo/runtime/internal/"
	if len(name) >= len(core) && name[:len(core)] == core {
		return false
	}
	const prefix = "runtime."
	if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
		name = name[len(prefix):]
		receiver := ""
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '.' {
				receiver, name = name[:i], name[i+1:]
				break
			}
		}
		if len(receiver) >= 3 && receiver[:2] == "(*" && receiver[len(receiver)-1] == ')' {
			receiver = receiver[2 : len(receiver)-1]
		}
		return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' && (len(receiver) == 0 || receiver[0] >= 'A' && receiver[0] <= 'Z')
	}
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			return true
		}
	}
	return false
}
