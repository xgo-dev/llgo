//go:build llgo && go1.27 && !wasm

package synctest_test

import (
	"context"
	"os"
	"os/exec"
	"reflect"
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
	for _, mode := range []string{"channel", "channel-poll", "channel-reflect-poll", "timer", "wait", "nested", "concurrent-wait", "cond", "waitgroup"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, self, "-test.run=^TestBubbleRejectedChild$", "-test.timeout=15s")
			cmd.Env = append(os.Environ(), "LLGO_SYNCTEST_REJECT="+mode)
			out, err := cmd.CombinedOutput()
			if ctx.Err() != nil || strings.Contains(string(out), "test timed out") {
				t.Fatalf("%s child timed out: %v\n%s", mode, err, out)
			}
			if mode == "waitgroup" && !strings.Contains(string(out), "WaitGroup.Add called from multiple synctest bubbles") {
				t.Fatalf("waitgroup child missed ownership rejection: %v\n%s", err, out)
			}
			if mode == "cond" && !strings.Contains(string(out), "wake from outside bubble") {
				t.Fatalf("cond child missed ownership rejection: %v\n%s", err, out)
			}
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
	case "channel-poll", "channel-reflect-poll":
		var ch chan int
		run(func() { ch = make(chan int) })
		if os.Getenv("LLGO_SYNCTEST_REJECT") == "channel-reflect-poll" {
			reflect.ValueOf(ch).TryRecv()
		} else {
			select {
			case <-ch:
			default:
			}
		}
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
		hold := make(chan struct{})
		go func() { (<-outside).Signal() }()
		run(func() {
			cond := sync.NewCond(new(sync.Mutex))
			go func() { cond.L.Lock(); cond.Wait(); cond.L.Unlock() }()
			synctest.Wait()
			outside <- cond
			<-hold
		})
	case "waitgroup":
		outside := make(chan *sync.WaitGroup)
		// An outside channel keeps the owner bubble active until the other
		// bubble tries Add. A bubble-owned nil wait can instead cause teardown
		// to remove the association before the rejected operation runs.
		hold := make(chan struct{})
		go func() { wg := <-outside; run(func() { wg.Add(1) }) }()
		run(func() { wg := new(sync.WaitGroup); wg.Add(1); outside <- wg; <-hold })
	}
	t.Fatal("invalid operation returned")
}
