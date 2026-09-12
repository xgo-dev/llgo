package returns

type Small struct {
	X, Y uint32
}

func Width(x, y uint32) Small {
	if x < y {
		return Small{x, y}
	}
	return Small{y, x}
}

func Pair(x uint64, y uint32) (uint64, uint32) {
	if x < uint64(y) {
		return x + 1, y
	}
	return x - 1, y + 1
}

func PairBool(x uint64, flag bool) (uint64, bool) {
	if x < 32 {
		return x + 1, flag
	}
	return x - 1, !flag
}
