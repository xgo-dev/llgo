package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

/* Handle types. */

type Check struct {
	Unused [120]byte
}

/* Function type */

// llgo:type C
type CheckCb func(Check *Check)

//go:linkname InitCheck C.uv_check_init
func InitCheck(loop *Loop, Check *Check) c.Int

// llgo:link (*Check).Start C.uv_check_start
func (Check *Check) Start(CheckCb CheckCb) c.Int { return 0 }

// llgo:link (*Check).Stop C.uv_check_stop
func (Check *Check) Stop() c.Int { return 0 }
