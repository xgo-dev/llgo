//go:build llgo && go1.27 && !wasm

package synctest_test

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	_ "unsafe"
)

//go:linkname run internal/synctest.Run
func run(func())

func TestBubbleRejections(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"channel", "timer", "wait", "nested", "concurrent-wait", "cond", "waitgroup"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(self, "-test.run=^TestBubbleRejectedChild$")
			cmd.Env = append(os.Environ(), "LLGO_SYNCTEST_REJECT="+mode)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "bubble") && mode != "concurrent-wait" {
				t.Fatalf("invalid %s accepted: %v\n%s", mode, err, out)
			}
		})
	}
}

func TestBubbleRejectedChild(t *testing.T) {
	switch os.Getenv("LLGO_SYNCTEST_REJECT") {
	case "":
		return
	case "channel":
		var ch chan int
		run(func() { ch = make(chan int, 1) })
		ch <- 1
	case "timer":
		var timer *time.Timer
		run(func() { timer = time.NewTimer(time.Hour) })
		timer.Stop()
	case "wait":
		synctest.Wait()
	case "nested":
		run(func() { run(func() {}) })
	case "concurrent-wait":
		run(func() { go synctest.Wait(); synctest.Wait() })
	case "cond":
		outside := make(chan *sync.Cond)
		go func() { (<-outside).Signal() }()
		run(func() {
			cond := sync.NewCond(new(sync.Mutex))
			go func() { cond.L.Lock(); cond.Wait(); cond.L.Unlock() }()
			synctest.Wait()
			outside <- cond
			<-make(chan struct{})
		})
	case "waitgroup":
		outside := make(chan *sync.WaitGroup)
		go func() { wg := <-outside; run(func() { wg.Add(1) }) }()
		run(func() { wg := new(sync.WaitGroup); wg.Add(1); outside <- wg; <-make(chan struct{}) })
	}
	t.Fatal("invalid operation returned")
}
