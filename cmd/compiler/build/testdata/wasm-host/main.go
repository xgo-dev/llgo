package main

import "crypto/rand"

func main() {
	var data [1]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
}
