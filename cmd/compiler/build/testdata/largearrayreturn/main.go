package main

const size = 128*1024 + 1

type params struct{ x, y, z int }

type versionLike struct {
	major, minor, patch     uint64
	pre, metadata, original string
}

func main() {
	a := f(1, 99)
	b := g(size-1, 98)
	c := h(size-1, 98)
	d := withAggregateParam(params{1, 2, 3})
	v := versionLike{major: 1, minor: 2, patch: 3, original: "v1.2.3"}.setPrerelease("beta")
	e := 0
	if v.pre == "beta" && v.original == "vbeta" {
		e = 1
	}
	println(a[1], b[1], c[1], a[size-1], b[size-1], c[size-1], d[0], e)
	println(f(1, 97)[1])
}

//go:noinline
func f(i, y int) (a [size]byte) {
	a[i] = byte(y)
	return
}

//go:noinline
func g(i, y int) [size]byte {
	var a [size]byte
	a[i] = byte(y)
	return a
}

//go:noinline
func h(i, y int) (a [size]byte) {
	a[i] = byte(y)
	return a
}

//go:noinline
func withAggregateParam(p params) (a [size]byte) {
	a[0] = byte(p.x + p.y + p.z)
	return
}

func (v versionLike) originalVPrefix() string {
	if v.original != "" && v.original[:1] == "v" {
		return "v"
	}
	return ""
}

func (v versionLike) setPrerelease(pre string) versionLike {
	vNext := v
	vNext.pre = pre
	vNext.original = v.originalVPrefix() + vNext.pre
	return vNext
}
