package spawn

func Value() int {
	done := make(chan int)
	go func() { done <- 42 }()
	return <-done
}
