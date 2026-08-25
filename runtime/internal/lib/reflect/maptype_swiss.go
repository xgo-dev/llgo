//go:build swissmap

package reflect

import "github.com/goplus/llgo/runtime/abi"

func mapTypeSetBucket(mt *mapType, k, e *abi.Type)     { mt.Group = nil }
func mapTypeSetKeySize(mt *mapType, n uint8)           {}
func mapTypeSetValueSize(mt *mapType, n uint8)         {}
func mapTypeSetBucketSize(mt *mapType, k, e *abi.Type) {}
