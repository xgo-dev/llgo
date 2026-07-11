//go:build darwin

package pthread

// Darwin defines pthread_key_t as unsigned long.
type Key uintptr
