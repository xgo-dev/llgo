//go:build !swissmap

package reflect

import "github.com/goplus/llgo/runtime/abi"

func mapTypeSetBucket(mt *mapType, k, e *abi.Type) { mt.Bucket = bucketOf(k, e) }
func mapTypeSetKeySize(mt *mapType, n uint8)       { mt.KeySize = n }
func mapTypeSetValueSize(mt *mapType, n uint8)     { mt.ValueSize = n }
func mapTypeSetBucketSize(mt *mapType, k, e *abi.Type) {
	mt.MapType.BucketSize = uint16(mt.Bucket.Size_)
}
