// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.
package main

// Eight parked goroutines are woken in a scrambled order. Each G reports its
// own number, so the aggregate is scheduling-order independent; the fiber
// scheduler additionally resumes them in FIFO wake order, which this golden
// deliberately does not depend on (hosted preemptive wakeups interleave).
func main() {
	var wake [8]chan int
	for i := range wake {
		wake[i] = make(chan int)
	}
	results := make(chan int, 8)
	for i := range wake {
		go func(n int) {
			<-wake[n]
			results <- n
		}(i)
	}
	for _, i := range [8]int{5, 2, 7, 0, 4, 1, 6, 3} {
		wake[i] <- 1
	}
	sum := 0
	for i := 0; i < 8; i++ {
		sum += <-results
	}
	if sum != 28 {
		println("bad fan sum", sum)
		return
	}
	println("wake fan ok")
}
