package kubelinkslicesa

import "slices"

func A() []string {
	return slices.AppendSeq([]string{"a"}, func(yield func(string) bool) {
		yield("aa")
	})
}
