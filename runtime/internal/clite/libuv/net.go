package libuv

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/net"
)

const (
	/* Used with uv_tcp_bind, when an IPv6 address is used. */
	TCP_IPV6ONLY TcpFlags = 1
)

/*
 * UDP support.
 */
const (
	/* Disables dual stack mode. */
	UDP_IPV6ONLY UdpFlags = 1
	/*
	 * Indicates message was truncated because read buffer was too small. The
	 * remainder was discarded by the OS. Used in uv_udp_recv_cb.
	 */
	UDP_PARTIAL UdpFlags = 2
	/*
	 * Indicates if SO_REUSEADDR will be set when binding the handle.
	 * This sets the SO_REUSEPORT socket flag on the BSDs and OS X. On other
	 * Unix platforms, it sets the SO_REUSEADDR flag.  What that means is that
	 * multiple threads or processes can bind to the same address without error
	 * (provided they all set the flag) but only the last one to bind will receive
	 * any traffic, in effect "stealing" the port from the previous listener.
	 */
	UDP_REUSEADDR UdpFlags = 4
	/*
	 * Indicates that the message was received by recvmmsg, so the buffer provided
	 * must not be freed by the recv_cb callback.
	 */
	UDP_MMSG_CHUNK UdpFlags = 8
	/*
	 * Indicates that the buffer provided has been fully utilized by recvmmsg and
	 * that it should now be freed by the recv_cb callback. When this flag is set
	 * in uv_udp_recv_cb, nread will always be 0 and addr will always be NULL.
	 */
	UDP_MMSG_FREE UdpFlags = 16
	/*
	 * Indicates if IP_RECVERR/IPV6_RECVERR will be set when binding the handle.
	 * This sets IP_RECVERR for IPv4 and IPV6_RECVERR for IPv6 UDP sockets on
	 * Linux. This stops the Linux kernel from suppressing some ICMP error
	 * messages and enables full ICMP error reporting for faster failover.
	 * This flag is no-op on platforms other than Linux.
	 */
	UDP_LINUX_RECVERR UdpFlags = 32
	/*
	 * Indicates that recvmmsg should be used, if available.
	 */
	UDP_RECVMMSG UdpFlags = 256
)

type TcpFlags c.Int

type UdpFlags c.Int

// ----------------------------------------------

/* Handle types. */

// TODO(spongehah): Handle
type Handle struct {
	Data   c.Pointer
	Unused [uvHandleSize - int(unsafe.Sizeof(uintptr(0)))]byte
}

// TODO(spongehah): Stream
type Stream struct {
	Data   c.Pointer
	Unused [256]byte
}

// TODO(spongehah): Tcp
type Tcp struct {
	Data   c.Pointer
	Unused [256]byte
}

// TODO(spongehah): Udp
type Udp struct {
	Unused [224]byte
}

/* Request types. */

// TODO(spongehah): Req
type Req struct {
	Unused [64]byte
}

// TODO(spongehah): UdpSend
type UdpSend struct {
	Unused [320]byte
}

// TODO(spongehah): Write
type Write struct {
	Data   c.Pointer
	Unused [184]byte
}

// TODO(spongehah): Connect
type Connect struct {
	Data   c.Pointer
	Unused [88]byte
}

// TODO(spongehah): GetAddrInfo
type GetAddrInfo struct {
	Unused [160]byte
}

// TODO(spongehah): GetNameInfo
type GetNameInfo struct {
	Unused [1320]byte
}

// TODO(spongehah): Shutdown
type Shutdown struct {
	Unused [80]byte
}

// ----------------------------------------------

/* Function type */

// llgo:type C
type CloseCb func(handle *Handle)

// llgo:type C
type ConnectCb func(req *Connect, status c.Int)

// llgo:type C
type UdpSendCb func(req *UdpSend, status c.Int)

// llgo:type C
type UdpRecvCb func(handle *Udp, nread c.Long, buf *Buf, addr *net.SockAddr, flags c.Uint)

// llgo:type C
type ReadCb func(stream *Stream, nread c.Long, buf *Buf)

// llgo:type C
type WriteCb func(req *Write, status c.Int)

// llgo:type C
type ConnectionCb func(server *Stream, status c.Int)

// llgo:type C
type ShutdownCb func(req *Shutdown, status c.Int)

// ----------------------------------------------

/* Handle related function and method */

//go:linkname HandleSize C.uv_handle_size
func HandleSize(handleType HandleType) uintptr

//go:linkname HandleTypeName C.uv_handle_type_name
func HandleTypeName(handleType HandleType) *c.Char

// llgo:link (*Handle).Ref C.uv_ref
func (handle *Handle) Ref() {}

// llgo:link (*Handle).Unref C.uv_unref
func (handle *Handle) Unref() {}

// llgo:link (*Handle).HasRef C.uv_has_ref
func (handle *Handle) HasRef() c.Int {
	return 0
}

// llgo:link (*Handle).GetType C.uv_handle_get_type
func (handle *Handle) GetType() HandleType {
	return 0
}

// llgo:link (*Handle).GetData C.uv_handle_get_data
func (handle *Handle) GetData() c.Pointer {
	return nil
}

// llgo:link (*Handle).GetLoop C.uv_handle_get_loop
func (handle *Handle) GetLoop() *Loop {
	return nil
}

// llgo:link (*Handle).SetData C.uv_handle_set_data
func (handle *Handle) SetData(data c.Pointer) {}

// llgo:link (*Handle).IsActive C.uv_is_active
func (handle *Handle) IsActive() c.Int {
	return 0
}

// llgo:link (*Handle).Close C.uv_close
func (handle *Handle) Close(closeCb CloseCb) {}

// llgo:link (*Handle).SendBufferSize C.uv_send_buffer_size
func (handle *Handle) SendBufferSize(value *c.Int) c.Int {
	return 0
}

// llgo:link (*Handle).RecvBufferSize C.uv_recv_buffer_size
func (handle *Handle) RecvBufferSize(value *c.Int) c.Int {
	return 0
}

// llgo:link (*Handle).Fileno C.uv_fileno
func (handle *Handle) Fileno(fd *OsFd) c.Int {
	return 0
}

// llgo:link (*Handle).IsClosing C.uv_is_closing
func (handle *Handle) IsClosing() c.Int {
	return 0
}

// llgo:link (*Handle).IsReadable C.uv_is_readable
func (handle *Handle) IsReadable() c.Int {
	return 0
}

// llgo:link (*Handle).IsWritable C.uv_is_writable
func (handle *Handle) IsWritable() c.Int {
	return 0
}

//go:linkname Pipe C.uv_pipe
func Pipe(fds [2]File, readFlags c.Int, writeFlags c.Int) c.Int {
	return 0
}

//go:linkname Socketpair C.uv_socketpair
func Socketpair(_type c.Int, protocol c.Int, socketVector [2]OsSock, flag0 c.Int, flag1 c.Int) c.Int {
	return 0
}

// ----------------------------------------------

/* Req related function and method */

//go:linkname ReqSize C.uv_req_size
func ReqSize(reqType ReqType) uintptr

//go:linkname TypeName C.uv_req_type_name
func TypeName(reqType ReqType) *c.Char

// llgo:link (*Req).GetData C.uv_req_get_data
func (req *Req) GetData() c.Pointer {
	return nil
}

// llgo:link (*Req).SetData C.uv_req_set_data
func (req *Req) SetData(data c.Pointer) {}

// llgo:link (*Req).GetType C.uv_req_get_type
func (req *Req) GetType() ReqType {
	return 0
}

// llgo:link (*Req).Cancel C.uv_cancel
func (req *Req) Cancel() c.Int {
	return 0
}

// ----------------------------------------------

/* Stream related function and method */

// llgo:link (*Stream).GetWriteQueueSize C.uv_stream_get_write_queue_size
func (stream *Stream) GetWriteQueueSize() uintptr {
	return 0
}

// llgo:link (*Stream).Listen C.uv_listen
func (stream *Stream) Listen(backlog c.Int, connectionCb ConnectionCb) c.Int {
	return 0
}

// llgo:link (*Stream).Accept C.uv_accept
func (server *Stream) Accept(client *Stream) c.Int {
	return 0
}

// llgo:link (*Stream).StartRead C.uv_read_start
func (stream *Stream) StartRead(allocCb AllocCb, readCb ReadCb) c.Int {
	return 0
}

// llgo:link (*Stream).StopRead C.uv_read_stop
func (stream *Stream) StopRead() c.Int {
	return 0
}

// llgo:link (*Write).Write C.uv_write
func (req *Write) Write(stream *Stream, bufs *Buf, nbufs c.Uint, writeCb WriteCb) c.Int {
	return 0
}

// llgo:link (*Write).Write2 C.uv_write2
func (req *Write) Write2(stream *Stream, bufs *Buf, nbufs c.Uint, sendStream *Stream, writeCb WriteCb) c.Int {
	return 0
}

// llgo:link (*Stream).TryWrite C.uv_try_write
func (stream *Stream) TryWrite(bufs *Buf, nbufs c.Uint) c.Int {
	return 0
}

// llgo:link (*Stream).TryWrite2 C.uv_try_write2
func (stream *Stream) TryWrite2(bufs *Buf, nbufs c.Uint, sendStream *Stream) c.Int {
	return 0
}

// llgo:link (*Stream).IsReadable C.uv_is_readable
func (stream *Stream) IsReadable() c.Int {
	return 0
}

// llgo:link (*Stream).IsWritable C.uv_is_writable
func (stream *Stream) IsWritable() c.Int {
	return 0
}

// llgo:link (*Stream).SetBlocking C.uv_stream_set_blocking
func (stream *Stream) SetBlocking(blocking c.Int) c.Int {
	return 0
}

//go:linkname StreamShutdown C.uv_shutdown
func StreamShutdown(shutdown *Shutdown, stream *Stream, shutdownCb ShutdownCb) c.Int {
	return 0
}

// ----------------------------------------------

/* Tcp related function and method */

//go:linkname InitTcp C.uv_tcp_init
func InitTcp(loop *Loop, tcp *Tcp) c.Int

//go:linkname InitTcpEx C.uv_tcp_init_ex
func InitTcpEx(loop *Loop, tcp *Tcp, flags c.Uint) c.Int

// llgo:link (*Tcp).Open C.uv_tcp_open
func (tcp *Tcp) Open(sock OsSock) c.Int {
	return 0
}

// llgo:link (*Tcp).Nodelay C.uv_tcp_nodelay
func (tcp *Tcp) Nodelay(enable c.Int) c.Int {
	return 0
}

// llgo:link (*Tcp).KeepAlive C.uv_tcp_keepalive
func (tcp *Tcp) KeepAlive(enable c.Int, delay c.Uint) c.Int {
	return 0
}

// llgo:link (*Tcp).SimultaneousAccepts C.uv_tcp_simultaneous_accepts
func (tcp *Tcp) SimultaneousAccepts(enable c.Int) c.Int {
	return 0
}

// llgo:link (*Tcp).Bind C.uv_tcp_bind
func (tcp *Tcp) Bind(addr *net.SockAddr, flags c.Uint) c.Int {
	return 0
}

// llgo:link (*Tcp).Getsockname C.uv_tcp_getsockname
func (tcp *Tcp) Getsockname(name *net.SockAddr, nameLen *c.Int) c.Int {
	return 0
}

// llgo:link (*Tcp).Getpeername C.uv_tcp_getpeername
func (tcp *Tcp) Getpeername(name *net.SockAddr, nameLen *c.Int) c.Int {
	return 0
}

// llgo:link (*Tcp).CloseReset C.uv_tcp_close_reset
func (tcp *Tcp) CloseReset(closeCb CloseCb) c.Int {
	return 0
}

// llgo:link (*Tcp).GetIoWatcherFd C.uv_tcp_get_io_watcher_fd
func (tcp *Tcp) GetIoWatcherFd() c.Int {
	return 0
}

//go:linkname TcpConnect C.uv_tcp_connect
func TcpConnect(req *Connect, tcp *Tcp, addr *net.SockAddr, connectCb ConnectCb) c.Int

// ----------------------------------------------

/* Udp related function and method */

//go:linkname InitUdp C.uv_udp_init
func InitUdp(loop *Loop, udp *Udp) c.Int

//go:linkname InitUdpEx C.uv_udp_init_ex
func InitUdpEx(loop *Loop, udp *Udp, flags c.Uint) c.Int

// llgo:link (*Udp).Open C.uv_udp_open
func (udp *Udp) Open(sock OsSock) c.Int {
	return 0
}

// llgo:link (*Udp).Bind C.uv_udp_bind
func (udp *Udp) Bind(addr *net.SockAddr, flags c.Uint) c.Int {
	return 0
}

// llgo:link (*Udp).Connect C.uv_udp_connect
func (udp *Udp) Connect(addr *net.SockAddr) c.Int {
	return 0
}

// llgo:link (*Udp).Getpeername C.uv_udp_getpeername
func (udp *Udp) Getpeername(name *net.SockAddr, nameLen *c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).Getsockname C.uv_udp_getsockname
func (udp *Udp) Getsockname(name *net.SockAddr, nameLen *c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).SetMembership C.uv_udp_set_membership
func (udp *Udp) SetMembership(multicastAddr *c.Char, interfaceAddr *c.Char, membership Membership) c.Int {
	return 0
}

// llgo:link (*Udp).SourceMembership C.uv_udp_set_source_membership
func (udp *Udp) SourceMembership(multicastAddr *c.Char, interfaceAddr *c.Char, sourceAddr *c.Char, membership Membership) c.Int {
	return 0
}

// llgo:link (*Udp).SetMulticastLoop C.uv_udp_set_multicast_loop
func (udp *Udp) SetMulticastLoop(on c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).SetMulticastTTL C.uv_udp_set_multicast_ttl
func (udp *Udp) SetMulticastTTL(ttl c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).SetMulticastInterface C.uv_udp_set_multicast_interface
func (udp *Udp) SetMulticastInterface(interfaceAddr *c.Char) c.Int {
	return 0
}

// llgo:link (*Udp).SetBroadcast C.uv_udp_set_broadcast
func (udp *Udp) SetBroadcast(on c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).SetTTL C.uv_udp_set_ttl
func (udp *Udp) SetTTL(ttl c.Int) c.Int {
	return 0
}

// llgo:link (*Udp).TrySend C.uv_udp_try_send
func (udp *Udp) TrySend(bufs *Buf, nbufs c.Uint, addr *net.SockAddr) c.Int {
	return 0
}

// llgo:link (*Udp).StartRecv C.uv_udp_recv_start
func (udp *Udp) StartRecv(allocCb AllocCb, recvCb UdpRecvCb) c.Int {
	return 0
}

// llgo:link (*Udp).UsingRecvmmsg C.uv_udp_using_recvmmsg
func (udp *Udp) UsingRecvmmsg() c.Int {
	return 0
}

// llgo:link (*Udp).StopRecv C.uv_udp_recv_stop
func (udp *Udp) StopRecv() c.Int {
	return 0
}

// llgo:link (*Udp).GetSendQueueSize C.uv_udp_get_send_queue_size
func (udp *Udp) GetSendQueueSize() uintptr {
	return 0
}

// llgo:link (*Udp).GetSendQueueCount C.uv_udp_get_send_queue_count
func (udp *Udp) GetSendQueueCount() uintptr {
	return 0
}

//go:linkname Send C.uv_udp_send
func Send(req *UdpSend, udp *Udp, bufs *Buf, nbufs c.Uint, addr *net.SockAddr, sendCb UdpSendCb) c.Int

// ----------------------------------------------

/* DNS related function and method */

//go:linkname Ip4Addr C.uv_ip4_addr
func Ip4Addr(ip *c.Char, port c.Int, addr *net.SockaddrIn) c.Int

//go:linkname Ip6Addr C.uv_ip6_addr
func Ip6Addr(ip *c.Char, port c.Int, addr *net.SockaddrIn6) c.Int

//go:linkname Ip4Name C.uv_ip4_name
func Ip4Name(src *net.SockaddrIn, dst *c.Char, size uintptr) c.Int

//go:linkname Ip6Name C.uv_ip6_name
func Ip6Name(src *net.SockaddrIn6, dst *c.Char, size uintptr) c.Int

//go:linkname IpName C.uv_ip_name
func IpName(src *net.SockAddr, dst *c.Char, size uintptr) c.Int

//go:linkname InetNtop C.uv_inet_ntop
func InetNtop(af c.Int, src c.Pointer, dst *c.Char, size uintptr) c.Int

//go:linkname InetPton C.uv_inet_pton
func InetPton(af c.Int, src *c.Char, dst c.Pointer) c.Int

// ----------------------------------------------

/* Getaddrinfo related function and method */

//go:linkname Getaddrinfo C.uv_getaddrinfo
func Getaddrinfo(loop *Loop, req *GetAddrInfo, getaddrinfoCb GetaddrinfoCb, node *c.Char, service *c.Char, hints *net.AddrInfo) c.Int

//go:linkname Freeaddrinfo C.uv_freeaddrinfo
func Freeaddrinfo(addrInfo *net.AddrInfo)

// ----------------------------------------------

/* Getnameinfo related function and method */

//go:linkname Getnameinfo C.uv_getnameinfo
func Getnameinfo(loop *Loop, req *GetNameInfo, getnameinfoCb GetnameinfoCb, addr *net.SockAddr, flags c.Int) c.Int
