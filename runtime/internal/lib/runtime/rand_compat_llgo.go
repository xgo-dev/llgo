package runtime

// Go 1.21 non-purego hash/maphash and external packages use this runtime symbol.
// Keep the same compatibility entry as upstream, backed by the platform rand.
func fastrand64() uint64 { return rand() }
