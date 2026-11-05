package assertmethods

type Value struct{}

func (Value) private() {}

//go:noinline
func Implements(v any) bool {
	_, ok := v.(interface{ private() })
	return ok
}
