// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

func main() {
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- "recovered"
			}
		}()
		panic("boom")
	}()
	println("recv", <-done)
	func() {
		defer func() { recover() }()
		panic(42)
	}()
	println("main ok")
	errch := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errch <- "guarded"
			}
		}()
		var s []int
		_ = s[3]
		errch <- "nofault"
	}()
	println("slice", <-errch)
}
