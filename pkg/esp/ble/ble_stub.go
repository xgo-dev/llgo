//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports the family's unsupported
// sentinel. Nothing here talks to hardware; vTaskDelay sleeps so a host-side
// caller of the API (a test, a smoke program) idles instead of spinning.
package ble

import (
	"time"
	"unsafe"
)

// errUnsupported is what every stub below returns: it lies above the
// esperr/ws/tls meta segments and below the nimble domain tag, so it can never
// be mistaken for a real code.
const errUnsupported = 0x7F00

func initHost() int32     { return errUnsupported }
func deinitHost() int32   { return errUnsupported }
// bondedPeers has no host-side counterpart: a hosted build has no bonds.
func bondedPeers(out *uint8, maxPeers int32) int32 { return 0 }

func ready() int32        { return 0 }
func printVersion() int32 { return errUnsupported }
func nowMs() uint32 {
	// A real millisecond clock, not a frozen 0: Next(>0) and waitOp measure
	// their deadlines against it, and on the host (gosched is a no-op, nothing
	// else advances time) a frozen clock would turn every bounded wait into an
	// infinite hot spin. The device build links the glue's esp_timer clock
	// instead; this stub is host-only.
	return uint32(time.Now().UnixMilli())
}
func scanStart(itvlMs, windowMs, passive, filterDup, durationMs int32, opID uint32) int32 {
	_, _, _, _, _, _ = itvlMs, windowMs, passive, filterDup, durationMs, opID
	return errUnsupported
}
func scanStop(opID uint32) int32 { _ = opID; return errUnsupported }
func connect(addr *uint8, addrType, timeoutMs int32, opID uint32) int32 {
	_, _, _, _ = addr, addrType, timeoutMs, opID
	return errUnsupported
}
func cancel(opID uint32) int32       { _ = opID; return errUnsupported }
func disconnect(connID uint32) int32 { _ = connID; return errUnsupported }
func poll(out *Event) int32          { _ = out; return 0 }

// ---- GATT client ----
//
// A host build has no stack, so every procedure reports the unsupported
// sentinel and the record accessors report an empty table. The Go API turns
// those into ErrUnsupported / ErrNotReady exactly as it does for a firmware
// built without the NimBLE host; nothing here invents a discovery result.

func discSvcs(connID, opID uint32) int32 {
	_, _ = connID, opID
	return errUnsupported
}

func discChrs(connID uint32, startHandle, endHandle uint16, opID uint32) int32 {
	_, _, _, _ = connID, startHandle, endHandle, opID
	return errUnsupported
}

func readAttr(connID uint32, handle uint16, opID uint32) int32 {
	_, _, _ = connID, handle, opID
	return errUnsupported
}

func writeAttr(connID uint32, handle uint16, data *uint8, n, mode int32, opID uint32) int32 {
	_, _, _, _, _, _ = connID, handle, data, n, mode, opID
	return errUnsupported
}

func svcCount(connID uint32) int32 { _ = connID; return 0 }

func svcAt(connID uint32, idx int32, out *svcRec) int32 {
	_, _, _ = connID, idx, out
	return 0
}

func chrCount(connID uint32) int32 { _ = connID; return 0 }

func chrAt(connID uint32, idx int32, out *chrRec) int32 {
	_, _, _ = connID, idx, out
	return 0
}

func mtuGet(connID uint32, out *int32) int32 {
	_, _ = connID, out
	return errUnsupported
}

// backendName reports "none": a host build carries no BT stack at all.
func backendName(buf *byte, n int32) int32 {
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

// gosched is a no-op on the host: the stub build never runs the poller, and
// the host scheduler has no fibers to hand off to.
func gosched() {}
