//go:build llgo && wasip1 && llgo.size.wasi

package syscall

import "unsafe"

// The compiler gives this guard a weak true default. A closed link can use
// the immutable lookup after proving that none of the environment writers run.
var llgoMutableEnvironment bool

// Closed links can omit the Go snapshot when no reachable code reads or
// modifies it. The host environment remains available to the C runtime.
var llgoEnvironmentNeeded bool

func llgoLoadEnvironment() []string {
	if llgoEnvironmentNeeded {
		return runtime_envs()
	}
	return nil
}

func llgoGetenvWithoutLock(key string) (string, bool) {
	if key == "" {
		return "", false
	}
	i, ok := env[key]
	if !ok {
		return "", false
	}
	// copyenv indexes the original entry up to its first equals sign.
	return envs[i][len(key)+1:], true
}

type llgoEnvironmentSlot struct {
	index   uint32
	keySize uint32
}

var llgoEnvironmentSlots []llgoEnvironmentSlot

func llgoLoadIndexedEnvironment() []string {
	if !llgoEnvironmentNeeded {
		return nil
	}
	entries := runtime_envs()
	if !llgoMutableEnvironment && len(entries) > 1 {
		llgoEnvironmentSlots = llgoIndexEnvironment(entries)
	}
	return entries
}

func llgoIndexEnvironment(entries []string) []llgoEnvironmentSlot {
	// Keep the table at most half full. Both the entry indices and key byte
	// counts fit in uint32 because the entries already fit in wasm32 memory.
	size := 2
	for size/2 < len(entries) {
		size *= 2
	}
	slots := make([]llgoEnvironmentSlot, size)
	for index, entry := range entries {
		for sep := 0; sep < len(entry); sep++ {
			if entry[sep] != '=' {
				continue
			}
			if sep != 0 {
				slot := llgoFindEnvironmentSlot(slots, entries, entry[:sep])
				if slots[slot].index == 0 {
					slots[slot] = llgoEnvironmentSlot{index: uint32(index + 1), keySize: uint32(sep)}
				}
			}
			break
		}
	}
	return slots
}

func llgoFindEnvironmentSlot(slots []llgoEnvironmentSlot, entries []string, key string) int {
	hash := llgoEnvironmentHash(unsafe.Pointer(unsafe.StringData(key)), 0, uintptr(len(key)))
	mask := len(slots) - 1
	for i := int(hash) & mask; ; i = (i + 1) & mask {
		slot := slots[i]
		if slot.index == 0 || int(slot.keySize) == len(key) && entries[slot.index-1][:slot.keySize] == key {
			return i
		}
	}
}

func llgoGetenvIndexed(key string) (string, bool) {
	if key == "" {
		return "", false
	}
	entries := envs
	if len(entries) == 1 {
		entry := entries[0]
		if len(entry) > len(key) && entry[len(key)] == '=' && entry[:len(key)] == key {
			// This is a possible match. Reject keys that include part of
			// the value, just as the index does by storing the first '='.
			for i := 0; i < len(key); i++ {
				if key[i] == '=' {
					return "", false
				}
			}
			return entry[len(key)+1:], true
		}
		return "", false
	}
	slots := llgoEnvironmentSlots
	if len(slots) == 0 {
		return "", false
	}
	slot := slots[llgoFindEnvironmentSlot(slots, entries, key)]
	if slot.index == 0 {
		return "", false
	}
	return entries[slot.index-1][slot.keySize+1:], true
}

// Use the same process-randomized hash as Go maps.
//
//go:linkname llgoEnvironmentHash runtime.memhash
//go:noescape
func llgoEnvironmentHash(unsafe.Pointer, uintptr, uintptr) uintptr
