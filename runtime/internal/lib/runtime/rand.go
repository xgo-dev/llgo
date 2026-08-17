package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/math"
)

//go:linkname c_rand C.rand
func c_rand() c.Int

func fastrand() uint32 {
	return uint32(c_rand())
}

func rand() uint64 {
	n := uint64(fastrand())
	n += 0xa0761d6478bd642f
	hi, lo := math.Mul64(n, n^0xe7037ed1a0b428db)
	return hi ^ lo
}

// randn is a fast reduction of rand() to [0, n).
// Do not change signature: used via linkname from other packages.
func randn(n uint32) uint32 {
	return uint32((uint64(uint32(rand())) * uint64(n)) >> 32)
}

//go:linkname os_fastrand os.fastrand
func os_fastrand() uint32 {
	return fastrand()
}

//go:linkname rand_fastrand64 math/rand.fastrand64
func rand_fastrand64() uint64 {
	return rand()
}

// sync_fastrandn is used by older stdlib sync implementations (Go 1.21).
//
//go:linkname sync_fastrandn sync.fastrandn
func sync_fastrandn(n uint32) uint32 {
	return randn(n)
}

// net_fastrandu is used by older stdlib net implementations.
//
//go:linkname net_fastrandu net.fastrandu
func net_fastrandu() uint {
	return uint(fastrand())
}
