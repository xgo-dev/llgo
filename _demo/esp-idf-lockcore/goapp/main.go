// Core-affinity acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main component;
// app_main calls the generated C entry, which hands control to the llgo fiber
// scheduler (runtime.RunMain).
//
// The probe drives runtime.LockCore over its whole contract on the IDF
// target, in order: the scheduler task boots pinned to HP core 0 (the
// sdkconfig deliberately leaves CONFIG_ESP_MAIN_TASK_AFFINITY at its CPU0
// default so the pins below really move the task), pin to core 1 verified
// through both the kernel's own getter (xTaskGetCoreID) and the executing
// core (esp_cpu_get_core_id via the demo glue) after a scheduling point, the
// same trip back to core 0, and the out-of-range rejection. Every step prints
// one `[go] lockcore` line; `[go] lockcore PASS` is the acceptance gate.
package main

import (
	"runtime"
	_ "unsafe" // the pull //go:linkname declarations below need it
)

// llgoFixStreams is the assembly helper merged into llgo_out.o
// (host/llgo_syms.S): it repoints clite's stdout/stderr/stdin handles at the
// ESP-IDF picolibc streams, because clite's baremetal init fopen fails under
// IDF's VFS. Must run after package init and before the first print.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// FreeRTOS bindings declared probe-side (the runtime package is internal;
// the idf_freertos.go comment documents this declaration pattern).
//
//go:linkname xTaskGetCurrentTaskHandle C.xTaskGetCurrentTaskHandle
func xTaskGetCurrentTaskHandle() uintptr

//go:linkname xTaskGetCoreID C.xTaskGetCoreID
func xTaskGetCoreID(handle uintptr) int32

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// llgoCurrentCore wraps esp_cpu_get_core_id for the probe's ground truth:
// the inline helper reads the executing hart, so a value read after a yield
// proves the task physically migrated, not just that the pin field flipped.
//
//go:linkname llgoCurrentCore C.llgoCurrentCore
func llgoCurrentCore() int32

// settleMs is the scheduling budget one migration gets: a vTaskDelay of two
// ticks at the default 100 Hz. On wake the task re-enters the ready state
// with its new pin, and the target core's scheduler picks it up.
const settleTicks = 2

// dec32 formats v in decimal (no strconv: the stdlib drags reflect machinery
// into a bare-metal image, the family rule every pkg/esp probe follows).
func dec32(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func main() {
	fixStreams()
	handle := xTaskGetCurrentTaskHandle()
	report := func(stage string) {
		println("[go] lockcore: " + stage +
			" pin=" + dec32(uint32(xTaskGetCoreID(handle))) +
			" core=" + dec32(uint32(llgoCurrentCore())))
	}

	report("start")

	// Pin the scheduler task to HP core 1 and watch it actually get there.
	if err := runtime.LockCore(1); err != nil {
		println("[go] lockcore: pin1 err " + err.Error())
		return
	}
	println("[go] lockcore: pin1 ok")
	vTaskDelay(settleTicks)
	report("after1")

	// And back to core 0: the pin is a switch, not a one-way latch.
	if err := runtime.LockCore(0); err != nil {
		println("[go] lockcore: pin0 err " + err.Error())
		return
	}
	println("[go] lockcore: pin0 ok")
	vTaskDelay(settleTicks)
	report("after0")

	// Out of range: the S31 has two HP cores, and the runtime refuses id 9
	// without touching the kernel.
	if err := runtime.LockCore(9); err == nil {
		println("[go] lockcore: range9 err <nil: refused range not reported>")
		return
	} else {
		println("[go] lockcore: range9 err " + err.Error())
	}

	println("[go] lockcore PASS")
	println("[go] lockcore done")
}
