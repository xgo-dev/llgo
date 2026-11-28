//go:build llgo && (amd64 || arm64 || 386) && !baremetal

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import "unsafe"

// These are compiler operations lowered by ssa/aes.go, which attaches the
// required CPU target-features and marks the containing kernel noinline so
// full LTO cannot hoist AES instructions across the runtime feature check.
//
//go:noescape
//go:linkname llgoAESRound llgo.aesRound
func llgoAESRound(op int, dst, src, key *byte)

//go:linkname llgoAESAlloca llgo.aesAlloca
func llgoAESAlloca(uintptr) unsafe.Pointer

// Must match the AESOp iota values in ssa/aes.go.
const (
	llgoAESXORBlock  = 6
	llgoAESHashRound = 7
	llgoAESHashLast  = 8
)

var (
	useAeshash  bool
	aeskeysched [128]byte
)

func aesHashEnvEnabled(hardware bool, env string) bool {
	enabled := hardware
	for env != "" {
		i := 0
		for i < len(env) && env[i] != ',' {
			i++
		}
		switch env[:i] {
		case "cpu.aes=off", "cpu.all=off":
			enabled = false
		case "cpu.aes=on", "cpu.all=on":
			enabled = hardware
		}
		if i == len(env) {
			break
		}
		env = env[i+1:]
	}
	return enabled
}

func initAESHash() {
	if !aesHashEnvEnabled(aesHashHardwareSupport(), CPUEnvironment()) {
		return
	}
	words := (*[16]uint64)(unsafe.Pointer(&aeskeysched[0]))
	for i := range words {
		words[i] = fastrand64()
	}
	useAeshash = true
}

// memhashAES hashes inputs larger than 16 bytes using hardware AES rounds
// (2 lanes for 17..32 B, 4 lanes for 33..64 B, and 8 parallel lanes for
// 65+ B). Callers must check useAeshash and s > 16 before invoking this
// function.
func memhashAES(p unsafe.Pointer, seed, s uintptr) uintptr {
	sv := (*[16]byte)(llgoAESAlloca(16))
	svWords := (*[2]uint64)(unsafe.Pointer(sv))
	svWords[0] = uint64(seed)
	svWords[1] = uint64(s)

	st := (*[8][16]byte)(llgoAESAlloca(128))
	d := (*[8][16]byte)(llgoAESAlloca(128))

	if s <= 32 {
		llgoAESRound(llgoAESHashRound, &st[0][0], &sv[0], &aeskeysched[0])
		llgoAESRound(llgoAESHashRound, &st[1][0], &sv[0], &aeskeysched[16])

		llgoAESRound(llgoAESHashRound, &d[0][0], (*byte)(p), &st[0][0])
		llgoAESRound(llgoAESHashRound, &d[1][0], (*byte)(add(p, s-16)), &st[1][0])

		llgoAESRound(llgoAESHashRound, &d[0][0], &d[0][0], &st[0][0])
		llgoAESRound(llgoAESHashRound, &d[1][0], &d[1][0], &st[1][0])

		llgoAESRound(llgoAESHashLast, &d[0][0], &d[0][0], &st[0][0])
		llgoAESRound(llgoAESHashLast, &d[1][0], &d[1][0], &st[1][0])

		llgoAESRound(llgoAESXORBlock, &d[0][0], &d[0][0], &d[1][0])
		return uintptr(*(*uint64)(unsafe.Pointer(&d[0][0])))
	}

	if s <= 64 {
		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESHashRound, &st[i][0], &sv[0], &aeskeysched[i*16])
		}
		llgoAESRound(llgoAESHashRound, &d[0][0], (*byte)(p), &st[0][0])
		llgoAESRound(llgoAESHashRound, &d[1][0], (*byte)(add(p, 16)), &st[1][0])
		llgoAESRound(llgoAESHashRound, &d[2][0], (*byte)(add(p, s-32)), &st[2][0])
		llgoAESRound(llgoAESHashRound, &d[3][0], (*byte)(add(p, s-16)), &st[3][0])

		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESHashRound, &d[i][0], &d[i][0], &st[i][0])
		}
		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESHashLast, &d[i][0], &d[i][0], &st[i][0])
		}
		llgoAESRound(llgoAESXORBlock, &d[0][0], &d[0][0], &d[2][0])
		llgoAESRound(llgoAESXORBlock, &d[1][0], &d[1][0], &d[3][0])
		llgoAESRound(llgoAESXORBlock, &d[0][0], &d[0][0], &d[1][0])
		return uintptr(*(*uint64)(unsafe.Pointer(&d[0][0])))
	}

	for i := 0; i < 8; i++ {
		llgoAESRound(llgoAESHashRound, &st[i][0], &sv[0], &aeskeysched[i*16])
	}

	if s <= 128 {
		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESHashRound, &d[i][0], (*byte)(add(p, uintptr(i*16))), &st[i][0])
		}
		tail := add(p, s-64)
		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESHashRound, &d[4+i][0], (*byte)(add(tail, uintptr(i*16))), &st[4+i][0])
		}
		for i := 0; i < 8; i++ {
			llgoAESRound(llgoAESHashRound, &d[i][0], &d[i][0], &st[i][0])
		}
		for i := 0; i < 8; i++ {
			llgoAESRound(llgoAESHashLast, &d[i][0], &d[i][0], &st[i][0])
		}
		for i := 0; i < 4; i++ {
			llgoAESRound(llgoAESXORBlock, &d[i][0], &d[i][0], &d[4+i][0])
		}
		llgoAESRound(llgoAESXORBlock, &d[0][0], &d[0][0], &d[2][0])
		llgoAESRound(llgoAESXORBlock, &d[1][0], &d[1][0], &d[3][0])
		llgoAESRound(llgoAESXORBlock, &d[0][0], &d[0][0], &d[1][0])
		return uintptr(*(*uint64)(unsafe.Pointer(&d[0][0])))
	}

	tail := add(p, s-128)
	// Start from the tail block mixed with the per-lane scrambled seed, then
	// run 2 full AES rounds per 128-byte block (keyed scramble + block absorb)
	// and 3 finalization rounds. Using the per-process secret aeskeysched lane
	// rather than st[i] itself as the round key preserves full diffusion on
	// both ARM (where AESE XORs the key before SubBytes) and x86 (where AESENC
	// XORs the key after MixColumns).
	for i := 0; i < 8; i++ {
		llgoAESRound(llgoAESHashRound, &st[i][0], (*byte)(add(tail, uintptr(i*16))), &st[i][0])
	}
	n := (s - 1) >> 7
	cur := p
	for ; n > 0; n-- {
		for i := 0; i < 8; i++ {
			llgoAESRound(llgoAESHashRound, &st[i][0], &st[i][0], &aeskeysched[i*16])
		}
		for i := 0; i < 8; i++ {
			llgoAESRound(llgoAESHashRound, &st[i][0], &st[i][0], (*byte)(add(cur, uintptr(i*16))))
		}
		cur = add(cur, 128)
	}
	for i := 0; i < 8; i++ {
		llgoAESRound(llgoAESHashRound, &st[i][0], &st[i][0], &aeskeysched[i*16])
	}
	for i := 0; i < 8; i++ {
		llgoAESRound(llgoAESHashRound, &st[i][0], &st[i][0], &aeskeysched[i*16])
	}
	for i := 0; i < 8; i++ {
		llgoAESRound(llgoAESHashLast, &st[i][0], &st[i][0], &aeskeysched[i*16])
	}
	for i := 0; i < 4; i++ {
		llgoAESRound(llgoAESXORBlock, &st[i][0], &st[i][0], &st[4+i][0])
	}
	llgoAESRound(llgoAESXORBlock, &st[0][0], &st[0][0], &st[2][0])
	llgoAESRound(llgoAESXORBlock, &st[1][0], &st[1][0], &st[3][0])
	llgoAESRound(llgoAESXORBlock, &st[0][0], &st[0][0], &st[1][0])
	return uintptr(*(*uint64)(unsafe.Pointer(&st[0][0])))
}
