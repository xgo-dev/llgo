//go:build llgo && go1.23 && windows && (amd64 || arm64)

package runtime

import "structs"

// Windows CONTEXT requires 16-byte alignment. Keep that storage contract on
// the owner without changing the platform record's fields or offsets.
type windowsFaultContextStorage struct {
	_     structs.Align16
	value windowsFaultContext
}

func (storage *windowsFaultContextStorage) context() *windowsFaultContext {
	return &storage.value
}
