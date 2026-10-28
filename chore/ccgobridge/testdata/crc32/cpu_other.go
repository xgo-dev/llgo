//go:build !amd64

package main

func requireAssemblyCPU() {
	panic("crc32 assembly probe requires amd64")
}
