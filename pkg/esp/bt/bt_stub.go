//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports the package's
// unsupported sentinel. Nothing here talks to hardware; vTaskDelay sleeps so
// a host-side caller of the API (a test, a smoke program) idles instead of
// spinning.
package bt

import (
	"time"
	"unsafe"
)

func glueABI() int32  { return glueABIVersion } // no contract on a host: let the stack check speak
func initBT() int32   { return codeUnsupport }
func deinitBT() int32 { return codeUnsupport }
func readyBT() int32  { return 0 }
func setName(name *uint8, n int32) int32 {
	_, _ = name, n
	return codeUnsupport
}
func setScanMode(discoverable int32) int32 { _ = discoverable; return codeUnsupport }

func sppInit() int32 { return codeUnsupport }
func sppStartSrv(scn int32, name *uint8, nameLen int32) int32 {
	_, _, _ = scn, name, nameLen
	return codeUnsupport
}
func sppDisconnect(handle uint32) int32 { _ = handle; return codeUnsupport }
func sppWrite(handle uint32, data *uint8, n int32) int32 {
	_, _, _ = handle, data, n
	return codeUnsupport
}

// sppRead reports "no record queued" (0) rather than the unsupported code:
// Read's contract is "0 = empty ring, retry", and on a host build the ring is
// honestly always empty, while the control calls above carry the unsupported
// answer.
func sppRead(handle uint32, dst *uint8, cap int32) int32 {
	_, _, _ = handle, dst, cap
	return 0
}

func poll(kind *int32, status *int32, handle *uint32, aux *int32, addr *uint8) int32 {
	_, _, _, _, _ = kind, status, handle, aux, addr
	return 0
}

// nowMs is a real millisecond clock, not a frozen 0: Next(>0) and every
// bounded loop measure their deadlines against it, and on the host (gosched
// is a no-op, nothing else advances time) a frozen clock would turn every
// bounded wait into an infinite hot spin. The device build links the glue's
// esp_timer clock instead; this stub is host-only.
func nowMs() uint32 { return uint32(time.Now().UnixMilli()) }

// backendName reports "none": a host build carries no BT stack at all.
func backendName(buf *uint8, n int32) int32 {
	const name = "none"
	if n <= 0 {
		return 0
	}
	dst := unsafe.Slice(buf, int(n))
	copied := copy(dst[:len(dst)-1], name)
	return int32(copied)
}

// vTaskDelay rests the host thread for the ticks a firmware build would yield
// to FreeRTOS. It is a real sleep rather than a no-op so that the package's
// poller goroutine idles instead of spinning when host-side code (a test, a
// `go run` smoke) exercises the API.
func vTaskDelay(ticks uint32) {
	time.Sleep(time.Duration(ticks) * 10 * time.Millisecond) // 100 Hz tick
}

// gosched is a no-op on the host: the host scheduler has no fibers to hand
// off to. Note the package's poller goroutine does run on a host build (the
// lazy start in Subscribe is unconditional) - it just gets its pacing from
// vTaskDelay's real sleep above, and never sees a mailbox record (the stub
// poll always reports an empty ring). Its fan-out list is only safe because
// the host tests never call Subscribe (they build subscriptions directly;
// see the bt_test.go file comment) - the cooperative-scheduler assumption
// that makes the unlocked subscribers slice safe on the device does not hold
// under the host's preemptive runtime.
func gosched() {}
