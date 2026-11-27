package main

func main() {
	for i := 0; i < 100; i++ {
		done := make(chan struct{})
		go func() {
			// No standard-library imports: the runtime's caller/panic state
			// alone must cause this worker to acquire a local context.
			for n := 0; n < 100; n++ {
				if recoveredPanic(n) != n {
					panic("worker recover lost panic value")
				}
			}
			close(done)
		}()
		<-done
	}
	println("wasi thread startup ok")
}

//go:noinline
func recoveredPanic(value int) (result any) {
	defer func() { result = recover() }()
	panic(value)
}
