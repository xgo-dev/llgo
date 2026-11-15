//go:build !nogc && !baremetal && !wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"

func setCollectorGCPercent(percent int32) int32 { return bdwgc.SetGCPercent(percent) }
