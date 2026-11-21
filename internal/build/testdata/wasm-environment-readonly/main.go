package main

import (
	"os"
	"sync"
)

func main() {
	var group sync.WaitGroup
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if value, ok := os.LookupEnv("LLGO_WASI_TEST_ENV"); !ok || value != "old=value" {
				panic("environment entry changed")
			}
			if _, ok := os.LookupEnv("LLGO_WASI_TEST_ENV=old"); ok {
				panic("lookup treated part of the value as a key")
			}
		}()
	}
	group.Wait()
	println("read-only environment lookup ok")
}
