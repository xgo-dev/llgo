package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

type Thread struct {
	Unused [8]byte
}

type ThreadOptions struct {
	flags     c.Uint
	stackSize uintptr
}

type Work struct {
	Unused [128]byte
}

// ----------------------------------------------

/* Function type */

// llgo:type C
type ThreadCb func(arg c.Pointer)

//llgo:type C
type WorkCb func(req *Work)

//llgo:type C
type AfterWorkCb func(req *Work, status c.Int)

// ----------------------------------------------

/* Thread related functions and method. */

//go:linkname ThreadEqual C.uv_thread_equal
func ThreadEqual(t1 *Thread, t2 *Thread) c.Int

//go:linkname ThreadGetCPU C.uv_thread_getcpu
func ThreadGetCPU() c.Int

//go:linkname ThreadSelf C.uv_thread_self
func ThreadSelf() Thread

// llgo:link (*Thread).Create C.uv_thread_create
func (t *Thread) Create(entry ThreadCb, arg c.Pointer) c.Int {
	return 0
}

// llgo:link (*Thread).CreateEx C.uv_thread_create_ex
func (t *Thread) CreateEx(entry ThreadCb, params *ThreadOptions, arg c.Pointer) c.Int {
	return 0
}

// llgo:link (*Thread).Join C.uv_thread_join
func (t *Thread) Join() c.Int {
	return 0
}

// llgo:link (*Thread).SetAffinity C.uv_thread_set_affinity
func (t *Thread) SetAffinity(cpuMask *c.Char, oldMask *c.Char, maskSize uintptr) c.Int {
	return 0
}

// llgo:link (*Thread).GetAffinity C.uv_thread_get_affinity
func (t *Thread) GetAffinity(cpuMask *c.Char, maskSize uintptr) c.Int {
	return 0
}

// ----------------------------------------------

/* Work related functions and method. */

//go:linkname QueueWork C.uv_queue_work
func QueueWork(loop *Loop, req *Work, workCb WorkCb, afterWorkCb AfterWorkCb) c.Int
