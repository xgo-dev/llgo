package main

import (
	"os"
	"sync"
)

var startupValue = os.Getenv("LLGO_WASI_TEST_ENV")

func main() {
	if startupValue != "old=value" {
		panic("environment unavailable during package initialization")
	}
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
			if _, ok := os.LookupEnv(""); ok {
				panic("lookup accepted an empty key")
			}
		}()
	}
	group.Wait()
	println("read-only environment lookup ok")
}
