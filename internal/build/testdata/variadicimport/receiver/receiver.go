package receiver

import _ "unsafe"

type Receiver int32

//llgo:link Receiver.Check C.review_check
func (Receiver) Check(__llgo_va_list ...any) int32 { return 0 }
