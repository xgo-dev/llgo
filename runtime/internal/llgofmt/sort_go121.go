//go:build go1.21

package llgofmt

import "slices"

func sortEntries[K comparable, V any](entries []entry[K, V], compare func(K, K) int) {
	slices.SortStableFunc(entries, func(a, b entry[K, V]) int {
		return compare(a.key, b.key)
	})
}
