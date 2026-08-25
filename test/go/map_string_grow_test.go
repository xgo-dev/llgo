package gotest

import (
	"strconv"
	"testing"
)

func TestMapStringLookupAfterGrow(t *testing.T) {
	const entries = 4096

	m := make(map[string]int)
	for i := 0; i < entries; i++ {
		m["key-"+strconv.Itoa(i)] = i
	}

	for i := 0; i < entries; i++ {
		key := "key-" + strconv.Itoa(i)
		if got, ok := m[key]; !ok || got != i {
			t.Fatalf("m[%q] = (%d, %v), want (%d, true)", key, got, ok, i)
		}
	}

	for i := 0; i < entries; i += 2 {
		delete(m, "key-"+strconv.Itoa(i))
	}

	if got, want := len(m), entries/2; got != want {
		t.Fatalf("len(m) = %d, want %d", got, want)
	}
	for key, value := range m {
		i, err := strconv.Atoi(key[len("key-"):])
		if err != nil || i%2 == 0 || value != i {
			t.Fatalf("unexpected entry after delete: %q: %d", key, value)
		}
	}
}
