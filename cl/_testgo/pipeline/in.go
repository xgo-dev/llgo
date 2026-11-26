// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

func main() {
	n := 6
	src := make(chan int)
	doubled := make(chan int)
	go func() {
		for i := 1; i <= n; i++ {
			src <- i
		}
		close(src)
	}()
	go func() {
		for v := range src {
			doubled <- v * 2
		}
		close(doubled)
	}()
	sum := 0
	for v := range doubled {
		sum += v
	}
	println("pipeline", sum)
	// worker pool: 3 workers, deterministic sum via results channel
	jobs := make(chan int, 8)
	results := make(chan int, 8)
	for w := 1; w <= 3; w++ {
		go func(id int) {
			for j := range jobs {
				results <- j*j + id*1000000
			}
		}(w)
	}
	go func() {
		for j := 1; j <= 5; j++ {
			jobs <- j
		}
		close(jobs)
	}()
	sq := 0
	for i := 0; i < 5; i++ {
		r := <-results
		sq += r % 1000000
	}
	println("pool", sq)
	// fan-in: merge two channels
	a := make(chan int, 2)
	b := make(chan int, 2)
	a <- 1
	a <- 2
	b <- 10
	close(a)
	close(b)
	merged := make(chan int)
	go func() {
		for v := range a {
			merged <- v
		}
		for v := range b {
			merged <- v
		}
		close(merged)
	}()
	total := 0
	for v := range merged {
		total += v
	}
	println("fanin", total)
}
