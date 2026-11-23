//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/tls_glue.c (the llgo_tls_* set) and main/net_glue.c (the shared
// socket/clock set). The same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design (host/make-artifact.sh lists them); the IDF component archive
// provides the definitions at the final link.
//
// The llgo_net_* externs deliberately duplicate the declarations in
// pkg/esp/net/net_idf.go: the packages stay mutually dependency-free, and
// both resolve against the ONE net_glue.c definition at the link - a TLS
// program simply links both glues (see the package doc). The shared set is
// exactly: socket, connect, poll, sock_error, shutdown, close, now_ms.
//
// All signatures are integer-and-pointer only (the pkg/esp/README.md
// boundary rule): the C-side mbedtls context travels as an opaque uintptr
// handle the GC never scans, and every function that produces a value takes
// an out-pointer, so errors can travel as returned codes.
package tls

import _ "unsafe" // for go:linkname

//go:linkname tlsNew C.llgo_tls_new
func tlsNew(fd int32, caPEM *byte, caLen int32, h *uintptr) int32

//go:linkname tlsHandstep C.llgo_tls_handstep
func tlsHandstep(h uintptr) int32

//go:linkname tlsRead C.llgo_tls_read
func tlsRead(h uintptr, buf *byte, length int32, n *int32) int32

//go:linkname tlsWrite C.llgo_tls_write
func tlsWrite(h uintptr, buf *byte, length int32, n *int32) int32

//go:linkname tlsCloseNotify C.llgo_tls_close_notify
func tlsCloseNotify(h uintptr) int32

//go:linkname tlsFree C.llgo_tls_free
func tlsFree(h uintptr)

// The shared socket/clock primitives, defined once in glue/net_glue.c.
//
//go:linkname netSocket C.llgo_net_socket
func netSocket(fd *int32) int32

//go:linkname netConnect C.llgo_net_connect
func netConnect(fd int32, ip uint32, port int32) int32

//go:linkname netPoll C.llgo_net_poll
func netPoll(fd int32, wantWrite int32, timeoutMs int32, ready *int32) int32

//go:linkname netSockError C.llgo_net_sock_error
func netSockError(fd int32, err *int32) int32

//go:linkname netShutdown C.llgo_net_shutdown
func netShutdown(fd int32, how int32) int32

//go:linkname netClose C.llgo_net_close
func netClose(fd int32) int32

//go:linkname nowMs C.llgo_net_now_ms
func nowMs() uint32
