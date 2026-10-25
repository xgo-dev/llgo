//go:build !windows || !386

package reflect

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// padFFIZeroSizedStructTail accounts for storage beyond natural alignment:
// two pointers followed by struct{} need a third word. Ordinary tail padding,
// as in { float64; float32; struct{} }, must remain implicit to preserve call
// classification.
func padFFIZeroSizedStructTail(typ *abi.Type, fields []*ffi.Type, off uintptr) []*ffi.Type {
	st := typ.StructType()
	if len(st.Fields) != 0 && st.Fields[len(st.Fields)-1].Typ.Size_ == 0 &&
		align(off, uintptr(typ.Align_)) < typ.Size_ {
		fields, _ = appendFFIPadding(fields, off, typ.Size_-off)
	}
	return fields
}
