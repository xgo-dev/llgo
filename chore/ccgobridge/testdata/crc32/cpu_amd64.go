package main

import "golang.org/x/sys/cpu"

func requireAssemblyCPU() {
	// These are the precise preconditions used by crc32 v1.3.0. Fail instead
	// of certifying its generic fallback as assembly execution.
	if !cpu.X86.HasSSE42 || !cpu.X86.HasSSE41 || !cpu.X86.HasPCLMULQDQ {
		panic("crc32 assembly probe requires SSE4.1, SSE4.2 and PCLMULQDQ")
	}
}
