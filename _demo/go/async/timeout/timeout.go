package timeout

import (
	"time"

	"github.com/xgo-dev/llgo/_demo/go/async/async"
)

func Timeout(d time.Duration) async.Future[async.Void] {
	return async.Async(func(resolve func(async.Void)) {
		go func() {
			time.Sleep(d)
			resolve(async.Void{})
		}()
	})
}
