package atomic

import _ "unsafe"

//go:linkname SwapInt32 llgo.atomicXchg
//go:linkname SwapInt64 llgo.atomicXchg
//go:linkname SwapUint32 llgo.atomicXchg
//go:linkname SwapUint64 llgo.atomicXchg
//go:linkname SwapUintptr llgo.atomicXchg
//go:linkname SwapPointer llgo.atomicXchg

//go:linkname CompareAndSwapInt32 llgo.atomicCmpXchgOK
//go:linkname CompareAndSwapInt64 llgo.atomicCmpXchgOK
//go:linkname CompareAndSwapUint32 llgo.atomicCmpXchgOK
//go:linkname CompareAndSwapUint64 llgo.atomicCmpXchgOK
//go:linkname CompareAndSwapUintptr llgo.atomicCmpXchgOK
//go:linkname CompareAndSwapPointer llgo.atomicCmpXchgOK

//go:linkname AddInt32 llgo.atomicAddReturnNew
//go:linkname AddInt64 llgo.atomicAddReturnNew
//go:linkname AddUint32 llgo.atomicAddReturnNew
//go:linkname AddUint64 llgo.atomicAddReturnNew
//go:linkname AddUintptr llgo.atomicAddReturnNew

//go:linkname LoadInt32 llgo.atomicLoad
//go:linkname LoadInt64 llgo.atomicLoad
//go:linkname LoadUint32 llgo.atomicLoad
//go:linkname LoadUint64 llgo.atomicLoad
//go:linkname LoadUintptr llgo.atomicLoad
//go:linkname LoadPointer llgo.atomicLoad

//go:linkname StoreInt32 llgo.atomicStore
//go:linkname StoreInt64 llgo.atomicStore
//go:linkname StoreUint32 llgo.atomicStore
//go:linkname StoreUint64 llgo.atomicStore
//go:linkname StoreUintptr llgo.atomicStore
//go:linkname StorePointer llgo.atomicStore

//go:linkname AndInt32 llgo.atomicAnd
//go:linkname AndInt64 llgo.atomicAnd
//go:linkname AndUint32 llgo.atomicAnd
//go:linkname AndUint64 llgo.atomicAnd
//go:linkname AndUintptr llgo.atomicAnd

//go:linkname OrInt32 llgo.atomicOr
//go:linkname OrInt64 llgo.atomicOr
//go:linkname OrUint32 llgo.atomicOr
//go:linkname OrUint64 llgo.atomicOr
//go:linkname OrUintptr llgo.atomicOr
