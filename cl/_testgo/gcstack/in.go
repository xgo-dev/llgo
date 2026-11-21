// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.
package main

type node struct {
	next *node
	val  int
}

// garbage produces one real heap allocation per call: the returned pointer
// forces buf to escape, so the compiler cannot lower it to an alloca. Callers
// drop the result, so each block dies before the next call and the loop keeps
// the collector running for real.
//
//go:noinline
func garbage(iter int) *[]byte {
	buf := make([]byte, 256)
	buf[0] = byte(iter)
	return &buf
}

// The list lives only in this goroutine's frames while it is parked: main
// never takes a reference, so the only thing keeping it alive across the
// collections below is the parked-fiber stack scan (scanBaremetalStackRoots'
// parked range on the RV32 baremetal scheduler). The G verifies the list
// itself after being woken, so a broken scan turns into a fail line instead
// of a false pass.
func main() {
	release := make(chan struct{})
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		var head *node
		for i := 0; i < 64; i++ {
			head = &node{next: head, val: i}
		}
		ready <- struct{}{} // handoff; this G parks on release right after
		<-release
		count, sum := 0, 0
		for n := head; n != nil; n = n.next {
			count++
			sum += n.val
		}
		if count != 64 || sum != 2016 {
			println("list corrupted", count, sum)
			close(done)
			return
		}
		println("gc stack ok")
		close(done)
	}()
	<-ready
	sink := 0
	for i := 0; i < 2000; i++ {
		garbage(i)
		sink += i & 1
	}
	_ = sink
	close(release)
	<-done // keep main parked until the woken G has verified the list
}
