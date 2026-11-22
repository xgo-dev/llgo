package main

import "time"

func main() {
	done := make(chan int, 1)
	time.AfterFunc(time.Millisecond, func() { done <- 42 })
	println("timer result", <-done)
}
