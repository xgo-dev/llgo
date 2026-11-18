package assertmethods

type Value struct{}

func (Value) private() int { return 17 }

//go:noinline
func Check(v any) (int, bool) {
	w, ok := v.(interface{ private() int })
	if !ok {
		return 0, false
	}
	return w.private(), true
}
