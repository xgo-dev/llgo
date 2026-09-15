//go:build wasm && !baremetal

package runtime

func signal_recv() uint32 {
	// No host signal can arrive on these profiles. Park the receiver rather
	// than returning a synthetic signal zero in a non-yielding receive loop.
	select {}
}
