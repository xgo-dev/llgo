package dep

const offset = 2

var Counter int

func Next() uint64 {
	Counter++
	return uint64(Counter)
}

func Sum(x, y uint64) uint64        { return x + y + offset }
func Split(x uint64) (uint64, bool) { return x + 1, x > 3 }

//go:noinline
func Kept(x uint64) uint64 { return x + 1 }

type Number uint64

func (n Number) Add(x uint64) uint64 { return uint64(n) + x + offset }

//go:noinline
func (n Number) Kept() uint64 { return uint64(n) + 1 }

type Box struct{ X uint64 }

func (b Box) Value() uint64 { return b.X }

func (b *Box) Read() uint64 {
	if b == nil {
		return 7
	}
	return b.X
}

func (b *Box) Address() *uint64       { return &b.X }
func (b *Box) Ignore(x uint64) uint64 { return x }
func (b *Box) Call(x uint64) uint64   { return (*b).Ignore(x) }

type Outer struct{ Box }
type Embedded struct{ *Box }
