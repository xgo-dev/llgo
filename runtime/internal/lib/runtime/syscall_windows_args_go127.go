//go:build windows && go1.27

package runtime

// Go1.27 uses dynamically sized Windows outgoing argument frames.
const windowsSyscallMaxArgs = 0
