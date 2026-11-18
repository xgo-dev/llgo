package value

type ScalarPair struct {
	Integer int64
	Float   float64
}

func (v ScalarPair) Shift(delta int64) ScalarPair {
	v.Integer += delta
	v.Float += float64(delta)
	return v
}

type Mixed struct {
	Integer int64
	Float   float64
	Pointer *int64
	Payload [8]uint64
}

func (v Mixed) Shift(delta int64) Mixed {
	v.Integer += delta
	v.Float += float64(delta)
	v.Payload[7] += uint64(delta)
	return v
}

type Summary struct {
	Integer int64
	Float   float64
	Pointer *int64
}

func (v Mixed) Summary() Summary {
	return Summary{v.Integer, v.Float, v.Pointer}
}
