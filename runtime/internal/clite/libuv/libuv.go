package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/net"
)

const (
	LLGoPackage = "link: $(pkg-config --libs libuv); -luv"
	LLGoFiles   = "$(pkg-config --cflags libuv): _wrap/libuv.c"
)

// ----------------------------------------------
const (
	RUN_DEFAULT RunMode = iota
	RUN_ONCE
	RUN_NOWAIT
)

const (
	LOOP_BLOCK_SIGNAL LoopOption = iota
	METRICS_IDLE_TIME
)

const (
	UV_LEAVE_GROUP Membership = iota
	UV_JOIN_GROUP
)

const (
	UNKNOWN_HANDLE HandleType = iota
	ASYNC
	CHECK
	FS_EVENT
	FS_POLL
	HANDLE
	IDLE
	NAMED_PIPE
	POLL
	PREPARE
	PROCESS
	STREAM
	TCP
	TIMER
	TTY
	UDP
	SIGNAL
	FILE
	HANDLE_TYPE_MAX
)

const (
	UNKNOWN_REQ ReqType = iota
	REQ
	CONNECT
	WRITE
	SHUTDOWN
	UDP_SEND
	FS
	WORK
	GETADDRINFO
	GETNAMEINFO
	RANDOM
	REQ_TYPE_PRIVATE
	REQ_TYPE_MAX
)

const (
	READABLE PollEvent = 1 << iota
	WRITABLE
	DISCONNECT
	PRIPRIORITIZED
)

type RunMode c.Int

type LoopOption c.Int

type Membership c.Int

type HandleType c.Int

type ReqType c.Int

type OsSock c.Int

type OsFd c.Int

type PollEvent c.Int

// ----------------------------------------------

/* Handle types. */

type Loop struct {
	Unused [0]byte
}

type Poll struct {
	Data   c.Pointer
	Unused [160]byte
}

/* Request types. */

type Buf struct {
	Base *c.Char
	Len  uintptr
} // ----------------------------------------------

/* Function type */

// llgo:type C
type MallocFunc func(size uintptr) c.Pointer

// llgo:type C
type ReallocFunc func(ptr c.Pointer, size uintptr) c.Pointer

// llgo:type C
type CallocFunc func(count uintptr, size uintptr) c.Pointer

// llgo:type C
type FreeFunc func(ptr c.Pointer)

// llgo:type C
type AllocCb func(handle *Handle, suggestedSize uintptr, buf *Buf)

// llgo:type C
type GetaddrinfoCb func(req *GetAddrInfo, status c.Int, res *net.AddrInfo)

// llgo:type C
type GetnameinfoCb func(req *GetNameInfo, status c.Int, hostname *c.Char, service *c.Char)

// llgo:type C
type WalkCb func(handle *Handle, arg c.Pointer)

// llgo:type C
type PollCb func(handle *Poll, status c.Int, events c.Int)

// ----------------------------------------------

//go:linkname Version C.uv_version
func Version() c.Uint

//go:linkname VersionString C.uv_version_string
func VersionString() *c.Char

//go:linkname LibraryShutdown C.uv_library_shutdown
func LibraryShutdown()

//go:linkname ReplaceAllocator C.uv_replace_allocator
func ReplaceAllocator(mallocFunc MallocFunc, reallocFunc ReallocFunc, callocFunc CallocFunc, freeFunc FreeFunc) c.Int

// ----------------------------------------------

/* Loop related functions and method. */

//go:linkname DefaultLoop C.uv_default_loop
func DefaultLoop() *Loop

//go:linkname LoopSize C.uv_loop_size
func LoopSize() uintptr

// llgo:link (*Loop).Run C.uv_run
func (loop *Loop) Run(mode RunMode) c.Int {
	return 0
}

// llgo:link (*Loop).Alive C.uv_loop_alive
func (loop *Loop) Alive() c.Int {
	return 0
}

// void uv_stop(uv_loop_t *loop)
//
// llgo:link (*Loop).Stop C.uv_stop
func (loop *Loop) Stop() {}

// llgo:link (*Loop).Close C.uv_loop_close
func (loop *Loop) Close() c.Int {
	return 0
}

// llgo:link (*Loop).Configure C.uv_loop_configure
func (loop *Loop) Configure(option LoopOption, arg c.Int) c.Int {
	return 0
}

// llgo:link LoopDefault C.uv_default_loop
func LoopDefault() *Loop {
	return nil
}

// llgo:link (*Loop).Delete C.uv_loop_delete
func (loop *Loop) Delete() c.Int {
	return 0
}

// llgo:link (*Loop).Fork C.uv_loop_fork
func (loop *Loop) Fork() c.Int {
	return 0
}

// llgo:link (*Loop).Init C.uv_loop_init
func (loop *Loop) Init() c.Int {
	return 0
}

// llgo:link LoopNew C.uv_loop_new
func LoopNew() *Loop {
	return nil
}

// llgo:link (*Loop).SetData C.uv_loop_set_data
func (loop *Loop) SetData(data c.Pointer) {
	return
}

// llgo:link (*Loop).GetData C.uv_loop_get_data
func (loop *Loop) GetData() c.Pointer {
	return nil
}

// llgo:link (*Loop).Now C.uv_now
func (loop *Loop) Now() c.UlongLong {
	return 0
}

// llgo:link (*Loop).UpdateTime C.uv_update_time
func (loop *Loop) UpdateTime() {
	// No return value needed for this method
}

// llgo:link (*Loop).BackendFd C.uv_backend_fd
func (loop *Loop) BackendFd() c.Int {
	return 0
}

// llgo:link (*Loop).BackendTimeout C.uv_backend_timeout
func (loop *Loop) BackendTimeout() c.Int {
	return 0
}

// llgo:link (*Loop).Walk C.uv_walk
func (loop *Loop) Walk(walkCb WalkCb, arg c.Pointer) {
	// No return value needed for this method
}

// ----------------------------------------------

/* Buf related functions and method. */

//go:linkname InitBuf C.uv_buf_init
func InitBuf(base *c.Char, len c.Uint) Buf

// ----------------------------------------------

/* Poll related function and method */

//go:linkname PollInit C.uv_poll_init
func PollInit(loop *Loop, handle *Poll, fd OsFd) c.Int

//go:linkname PollInitSocket C.uv_poll_init_socket
func PollInitSocket(loop *Loop, handle *Poll, socket c.Int) c.Int

// llgo:link (*Poll).Start C.uv_poll_start
func (handle *Poll) Start(events c.Int, cb PollCb) c.Int {
	return 0
}

// llgo:link (*Poll).Stop C.uv_poll_stop
func (handle *Poll) Stop() c.Int {
	return 0
}
