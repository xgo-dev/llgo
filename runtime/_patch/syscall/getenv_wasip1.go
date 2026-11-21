//go:build llgo && wasip1 && llgo.size.wasi

package syscall

// The compiler gives this guard a weak true default. A closed link can use
// the immutable lookup after proving that none of the environment writers run.
var llgoMutableEnvironment bool

// Closed links can omit the Go snapshot when no reachable code reads or
// modifies it. The host environment remains available to the C runtime.
var llgoEnvironmentNeeded bool

func llgoLoadEnvironment() []string {
	if llgoEnvironmentNeeded {
		return runtime_envs()
	}
	return nil
}

func llgoGetenvWithoutLock(key string) (string, bool) {
	if key == "" {
		return "", false
	}
	i, ok := env[key]
	if !ok {
		return "", false
	}
	// copyenv indexes the original entry up to its first equals sign.
	return envs[i][len(key)+1:], true
}
