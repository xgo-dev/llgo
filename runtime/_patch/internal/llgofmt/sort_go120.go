//go:build llgo && wasip1 && wasm && llgo.fmt.value && !go1.21

package llgofmt

import "sort"

type sortableEntries[K comparable, V any] struct {
	entries []entry[K, V]
	compare func(K, K) int
}

func (s sortableEntries[K, V]) Len() int { return len(s.entries) }
func (s sortableEntries[K, V]) Less(i, j int) bool {
	return s.compare(s.entries[i].key, s.entries[j].key) < 0
}
func (s sortableEntries[K, V]) Swap(i, j int) {
	s.entries[i], s.entries[j] = s.entries[j], s.entries[i]
}

func sortEntries[K comparable, V any](entries []entry[K, V], compare func(K, K) int) {
	sort.Stable(sortableEntries[K, V]{entries, compare})
}
