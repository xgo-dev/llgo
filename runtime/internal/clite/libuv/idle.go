package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

/* Handle types. */

type Idle struct {
	Unused [120]byte
}

/* Function type */

// llgo:type C
type IdleCb func(idle *Idle)

//go:linkname InitIdle C.uv_idle_init
func InitIdle(loop *Loop, idle *Idle) c.Int

// llgo:link (*Idle).Start C.uv_idle_start
func (idle *Idle) Start(idleCb IdleCb) c.Int { return 0 }

// llgo:link (*Idle).Stop C.uv_idle_stop
func (idle *Idle) Stop() c.Int { return 0 }
