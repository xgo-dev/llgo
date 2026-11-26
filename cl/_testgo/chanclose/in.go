// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.
package main

// Close wakes a parked receiver with the zero value and ok=false, whichever
// ran first: an already-closed channel completes the receive without parking.
func main() {
	c := make(chan int)
	done := make(chan struct{})
	go func() {
		v, ok := <-c
		println("recv", v, ok)
		close(done)
	}()
	close(c)
	<-done
	println("closed ok")
}
