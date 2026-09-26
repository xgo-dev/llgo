package main

func main() {
	for i := 0; i < 100; i++ {
		done := make(chan struct{})
		go func() { close(done) }()
		<-done
	}
	println("wasi thread startup ok")
}
