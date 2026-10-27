package derived

import "github.com/xgo-dev/llgo/test/llgoext/nativerepr/native"

type First native.Record
type Copy First
type Alias = native.Record
type ThroughAlias Alias
type Instance native.Generic[uint64]
type Recursive native.Recursive
