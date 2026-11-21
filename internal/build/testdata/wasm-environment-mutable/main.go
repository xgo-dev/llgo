package main

import "os"

func main() {
	const key = "LLGO_WASI_TEST_ENV"
	if os.Getenv(key) != "old=value" {
		panic("initial environment entry changed")
	}
	if err := os.Setenv(key, "next"); err != nil || os.Getenv(key) != "next" {
		panic("Setenv lost the update")
	}
	if err := os.Unsetenv(key); err != nil {
		panic(err)
	}
	if _, ok := os.LookupEnv(key); ok {
		panic("Unsetenv lost the update")
	}
	os.Clearenv()
	if len(os.Environ()) != 0 {
		panic("Clearenv lost the update")
	}
	println("mutable environment lookup ok")
}
