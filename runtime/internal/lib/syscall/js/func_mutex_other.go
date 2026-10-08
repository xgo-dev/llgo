//go:build js && wasm && (!llgo || !llgo.wasm.workers)

package js

import "sync"

type funcMutex = sync.Mutex
