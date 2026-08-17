package main

import (
	"time"

	"github.com/xgo-dev/llgo/_demo/go/async/async"
	"github.com/xgo-dev/llgo/_demo/go/async/timeout"
)

func Sleep(i int, d time.Duration) async.Future[int] {
	return async.Async(func(resolve func(int)) {
		timeout.Timeout(d).Then(func(async.Void) {
			resolve(i)
		})
	})
}

func main() {
	async.Run(async.Async(func(resolve func(async.Void)) {
		println("read file")
		defer resolve(async.Void{})
	}))
}
