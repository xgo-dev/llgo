package runtime

// LockCore pins the execution context that carries the Go scheduler to
// hardware core id.
//
// This file exists for the type-check surface only: llgo's loader sees
// GOROOT's runtime plus these patch files, while every lowering and the real
// implementations come from the alternative runtime package
// (runtime/internal/lib/runtime). The body below never runs under llgo - the
// alternative runtime replaces the package wholesale - so it only has to
// fail loudly if that invariant ever breaks.
//
// The //llgo:driver directive also feeds this file to the build driver's
// -overlay view (the source toolchain that export-checks user code), because
// user programs calling LockCore must type-check there too.
//
// Target semantics (implemented in the alternative runtime):
//   - esp32s31-idf: pin the FreeRTOS task hosting the fiber scheduler to HP
//     core id; the pin reads back through the kernel getter before success.
//   - non-IDF RV32 baremetal: single-hart scheduler, always locked.
//   - everything else: explicit error, never a silent pretend-pin.
//
//llgo:driver
func LockCore(id int) error {
	panic("runtime: LockCore not implemented")
}
