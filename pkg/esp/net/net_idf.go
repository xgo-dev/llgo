//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/net_glue.c — the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design (host/make-artifact.sh lists them); the IDF component archive
// provides the definitions at the final link.
//
// All signatures are integer-and-pointer only (the pkg/esp/README.md
// boundary rule): ports cross as host-order int32, and every function that
// produces a value takes an out-pointer, so errors can travel as returned
// lwip errno values.
package net

import _ "unsafe" // for go:linkname

//go:linkname netSocket C.llgo_net_socket
func netSocket(fd *int32) int32

//go:linkname netConnect C.llgo_net_connect
func netConnect(fd int32, ip uint32, port int32) int32

//go:linkname netPoll C.llgo_net_poll
func netPoll(fd int32, wantWrite int32, timeoutMs int32, ready *int32) int32

//go:linkname netSockError C.llgo_net_sock_error
func netSockError(fd int32, err *int32) int32

//go:linkname netRecv C.llgo_net_recv
func netRecv(fd int32, buf *uint8, length int32, n *int32) int32

//go:linkname netSend C.llgo_net_send
func netSend(fd int32, buf *uint8, length int32, n *int32) int32

//go:linkname netShutdown C.llgo_net_shutdown
func netShutdown(fd int32, how int32) int32

//go:linkname netClose C.llgo_net_close
func netClose(fd int32) int32

//go:linkname netBindListen C.llgo_net_bind_listen
func netBindListen(fd int32, port int32, backlog int32) int32

//go:linkname netAccept C.llgo_net_accept
func netAccept(fd int32, connFd *int32) int32

//go:linkname netResolve C.llgo_net_resolve
func netResolve(host *uint8, hostLen int32, ip *uint32) int32

//go:linkname nowMs C.llgo_net_now_ms
func nowMs() uint32
