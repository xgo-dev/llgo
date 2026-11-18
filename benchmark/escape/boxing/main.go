package main

import (
	"encoding/json"
	"os"
	"runtime"
	"strconv"
)

type host func(*uint64, []uint64) error
type binding struct{ callback host }

//go:noinline
func assertFunction(callback any, counter *uint64, stack []uint64) error {
	f := callback.(host)
	return f(counter, stack)
}

//go:noinline
func assertRecord(callback any, counter *uint64, stack []uint64) error {
	return callback.(*binding).callback(counter, stack)
}

//go:noinline
func directFunction(callback host, counter *uint64, stack []uint64) error {
	return callback(counter, stack)
}

//go:noinline
func directRecord(callback *binding, counter *uint64, stack []uint64) error {
	return callback.callback(counter, stack)
}

//go:noinline
func boxFunction(callback host, counter *uint64, stack []uint64) error {
	var boxed any = callback
	return assertFunction(boxed, counter, stack)
}

type row struct {
	Name       string `json:"name"`
	Calls      int    `json:"calls"`
	Checksum   uint64 `json:"checksum"`
	TotalAlloc uint64 `json:"totalAlloc"`
}

func measure(name string, fn func() error, counter *uint64, bias uint64) row {
	const calls = 100000
	for range 64 {
		if err := fn(); err != nil {
			panic(err)
		}
	}
	*counter = 0
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range calls {
		if err := fn(); err != nil {
			panic(err)
		}
	}
	runtime.ReadMemStats(&after)
	if *counter != uint64(calls)*3*bias {
		panic("incorrect checksum")
	}
	return row{name, calls, *counter, after.TotalAlloc - before.TotalAlloc}
}
func main() {
	bias := uint64(3)
	if len(os.Args) > 1 {
		var err error
		bias, err = strconv.ParseUint(os.Args[1], 10, 64)
		if err != nil {
			panic(err)
		}
	}
	callback := host(func(counter *uint64, stack []uint64) error { *counter += (stack[0] + stack[1]) * bias; return nil })
	record := &binding{callback: callback}
	var boxed any = callback
	var boxedRecord any = record
	var counter uint64
	stack := []uint64{1, 2}
	rows := []row{
		measure("box-function", func() error { return boxFunction(callback, &counter, stack) }, &counter, bias),
		measure("assert-preboxed-function", func() error { return assertFunction(boxed, &counter, stack) }, &counter, bias),
		measure("assert-preboxed-record", func() error { return assertRecord(boxedRecord, &counter, stack) }, &counter, bias),
		measure("assert-record", func() error { return assertRecord(record, &counter, stack) }, &counter, bias),
		measure("direct-function", func() error { return directFunction(callback, &counter, stack) }, &counter, bias),
		measure("direct-record", func() error { return directRecord(record, &counter, stack) }, &counter, bias),
	}
	if err := json.NewEncoder(os.Stdout).Encode(rows); err != nil {
		panic(err)
	}
}
