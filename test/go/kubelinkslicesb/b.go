package kubelinkslicesb

import "slices"

func B() []string {
	return slices.AppendSeq([]string{"b"}, func(yield func(string) bool) {
		yield("bb")
	})
}
