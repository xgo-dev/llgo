//go:build windows && !go1.27

package runtime

// Preserve the selected older SDK's original SyscallN boundary.
const windowsSyscallMaxArgs = 42
