package generic

func Launch[T any](value T) T {
	done := make(chan T)
	go func() { done <- value }()
	return <-done
}
