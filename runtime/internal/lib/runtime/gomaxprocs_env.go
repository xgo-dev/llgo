//go:build !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

func gomaxprocsEnv() string {
	if value := cliteos.Getenv(c.AllocaCStr("GOMAXPROCS")); value != nil {
		return c.GoString(value)
	}
	return ""
}
