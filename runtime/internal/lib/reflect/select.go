package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// A SelectDir describes the communication direction of a select case.
type SelectDir int

const (
	_             SelectDir = iota
	SelectSend              // case Chan <- Send
	SelectRecv              // case <-Chan:
	SelectDefault           // default
)

// A SelectCase describes a single case in a select operation.
type SelectCase struct {
	Dir  SelectDir // direction of case
	Chan Value     // channel to use (for send or receive)
	Send Value     // value to send (for send)
}

// Select executes a select operation described by the list of cases.
func Select(cases []SelectCase) (chosen int, recv Value, recvOK bool) {
	if len(cases) > 65536 {
		panic("reflect.Select: too many cases (max 65536)")
	}
	if len(cases) == 0 {
		select {}
	}

	type recvMeta struct {
		v     Value
		isRec bool
	}
	ops := make([]runtime.ChanOp, 0, len(cases))
	indexes := make([]int, 0, len(cases))
	metas := make([]recvMeta, 0, len(cases))
	defaultIdx := -1

	for i, c := range cases {
		switch c.Dir {
		default:
			panic("reflect.Select: invalid Dir")
		case SelectDefault:
			if defaultIdx >= 0 {
				panic("reflect.Select: multiple default cases")
			}
			if c.Chan.IsValid() {
				panic("reflect.Select: default case has Chan value")
			}
			if c.Send.IsValid() {
				panic("reflect.Select: default case has Send value")
			}
			defaultIdx = i
		case SelectSend:
			ch := c.Chan
			if !ch.IsValid() {
				// Zero Value channel disables this case.
				continue
			}
			ch.mustBe(Chan)
			ch.mustBeExported()
			tt := (*chanType)(unsafe.Pointer(ch.typ()))
			if ChanDir(tt.Dir)&SendDir == 0 {
				panic("reflect.Select: SendDir case using recv-only channel")
			}
			v := c.Send
			if !v.IsValid() {
				panic("reflect.Select: SendDir case missing Send value")
			}
			v.mustBeExported()
			v = v.assignTo("reflect.Select", tt.Elem, nil)
			var p unsafe.Pointer
			if v.flag&flagIndir != 0 {
				p = v.ptr
			} else {
				p = unsafe.Pointer(&v.ptr)
			}
			escapes(p)
			ops = append(ops, runtime.ChanOp{
				C:    (*runtime.Chan)(ch.pointer()),
				Val:  p,
				Size: int32(tt.Elem.Size()),
				Send: true,
			})
			indexes = append(indexes, i)
			metas = append(metas, recvMeta{})
		case SelectRecv:
			if c.Send.IsValid() {
				panic("reflect.Select: RecvDir case has Send value")
			}
			ch := c.Chan
			if !ch.IsValid() {
				// Zero Value channel disables this case.
				continue
			}
			ch.mustBe(Chan)
			ch.mustBeExported()
			tt := (*chanType)(unsafe.Pointer(ch.typ()))
			if ChanDir(tt.Dir)&RecvDir == 0 {
				panic("reflect.Select: RecvDir case using send-only channel")
			}
			t := tt.Elem
			// Always receive into dedicated storage. If we receive into &rv.ptr
			// for direct-interface types, metas append would copy rv before
			// channel receive writes back, leaving stale zero values.
			p := unsafe_New(t)
			rv := Value{t, p, flag(t.Kind()) | flagIndir}
			ops = append(ops, runtime.ChanOp{
				C:    (*runtime.Chan)(ch.pointer()),
				Val:  p,
				Size: int32(t.Size()),
				Send: false,
			})
			indexes = append(indexes, i)
			metas = append(metas, recvMeta{v: rv, isRec: true})
		}
	}

	if defaultIdx >= 0 {
		if len(ops) == 0 {
			return defaultIdx, Value{}, false
		}
		isel, ok, tryOK := runtime.TrySelect(ops...)
		if !tryOK {
			return defaultIdx, Value{}, false
		}
		chosen = indexes[isel]
		if metas[isel].isRec {
			return chosen, metas[isel].v, ok
		}
		return chosen, Value{}, false
	}

	if len(ops) == 0 {
		// All cases are nil-channel operations and no default exists.
		select {}
	}

	isel, ok := runtime.Select(ops...)
	chosen = indexes[isel]
	if metas[isel].isRec {
		return chosen, metas[isel].v, ok
	}
	return chosen, Value{}, false
}
