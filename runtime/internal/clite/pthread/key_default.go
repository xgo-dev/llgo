//go:build !darwin

package pthread

import c "github.com/goplus/llgo/runtime/internal/clite"

type Key c.Uint
