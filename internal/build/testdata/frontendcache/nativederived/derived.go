package nativederived

import "example.com/frontprobe/nativebase"

type First nativebase.Record
type Record First

//go:noinline
func Local[T any](value T) bool {
	type Copy nativebase.Record
	frame := struct {
		Value    nativebase.Record
		Sentinel uint64
	}{Value: nativebase.Record{Tail: 7}, Sentinel: 123456}
	p := (*Copy)(&frame.Value)
	p.Tail = 99
	return frame.Value.Tail == 99 && frame.Sentinel == 123456
}
