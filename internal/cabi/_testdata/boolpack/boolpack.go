package boolpack

// Pack matches C struct { _Bool a; char b; _Bool c; } (size 3, offsets 0,1,2).
type Pack struct {
	A bool
	B byte
	C bool
}

func Echo(p Pack) Pack { return p }

func Not(x bool) bool { return !x }

func Fill(dst *[256]bool, v bool) {
	for i := range dst {
		dst[i] = v
	}
}
