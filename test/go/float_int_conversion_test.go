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

package gotest

import (
	"os"
	"path/filepath"
	"testing"
)

const floatIntConversionProbe = `package main

import (
	"fmt"
	"math"
)

//go:noinline
func id[T any](x T) T {
	return x
}

func check[T comparable](name string, got, want T) {
	if got != want {
		panic(fmt.Sprintf("%s: got %v want %v", name, got, want))
	}
}

func main() {
	one := id(1.0)
	minusOne32 := id(float32(-1))
	minusOne64 := id(float64(-1))
	p32Plus := id(float32(2147483647 + 4096 + 1))
	p64Plus := id(float64(9223372036854775807 + 4096 + 1))
	u32NearMax := id(float32(4294967295.1))
	u32Over := id(float32(5294967295.1))
	u32NegLarge := id(float32(-1294967295.1))
	u32NegSmall := id(float32(-1.1))
	n32Minus := id(float32(-2147483648 - 4096))
	n64Minus := id(float64(-9223372036854775808 - 4096))
	inf32 := id(float32(one / 0))
	inf64 := id(float64(one / 0))
	ninf32 := id(float32(-one / 0))
	ninf64 := id(float64(-one / 0))
	nan32 := id(float32(math.NaN()))
	nan64 := id(math.NaN())

	check("int32 high f32", int32(p32Plus), int32(2147483647))
	check("int32 high f64", int32(p64Plus), int32(2147483647))
	check("int32 low f32", int32(n32Minus), int32(-2147483648))
	check("int32 low f64", int32(n64Minus), int32(-2147483648))
	check("int32 inf", int32(inf32), int32(2147483647))
	check("int32 inf64", int32(inf64), int32(2147483647))
	check("int32 ninf", int32(ninf32), int32(-2147483648))
	check("int32 ninf64", int32(ninf64), int32(-2147483648))
	check("int32 nan", int32(nan32), int32(0))
	check("int32 nan64", int32(nan64), int32(0))

	check("int64 high f64", int64(p64Plus), int64(9223372036854775807))
	check("int64 low f64", int64(n64Minus), int64(-9223372036854775808))
	check("int64 inf", int64(inf64), int64(9223372036854775807))
	check("int64 ninf", int64(ninf64), int64(-9223372036854775808))
	check("int64 nan", int64(nan64), int64(0))

	check("uint32 negative f32", uint32(minusOne32), ^uint32(0))
	check("uint32 negative f64", uint32(minusOne64), ^uint32(0))
	check("uint32 high f64", uint32(p64Plus), ^uint32(0))
	check("uint32 near max f32", uint32(u32NearMax), uint32(0))
	check("uint32 over f32", uint32(u32Over), uint32(1000000000))
	check("uint32 negative large f32", uint32(u32NegLarge), uint32(3000000000))
	check("uint32 negative small f32", uint32(u32NegSmall), ^uint32(0))
	check("uint32 inf", uint32(inf64), ^uint32(0))
	check("uint32 ninf", uint32(ninf64), uint32(0))
	check("uint32 nan", uint32(nan64), uint32(0))

	check("uint64 negative", uint64(minusOne64), uint64(0))
	check("uint64 inf", uint64(inf64), ^uint64(0))
	check("uint64 ninf", uint64(ninf64), uint64(0))
	check("uint64 nan", uint64(nan64), uint64(0))

	check("uint8 negative wraps", uint8(minusOne64), uint8(255))
	check("uint8 high clamps then truncates", uint8(p32Plus), uint8(255))
	check("uint8 ninf truncates", uint8(ninf64), uint8(0))
	check("int8 high truncates", int8(p32Plus), int8(-1))
	check("int8 ninf truncates", int8(ninf64), int8(0))
}
`

func TestFloatToIntegerConversionSemantics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	if err := os.WriteFile(file, []byte(floatIntConversionProbe), 0644); err != nil {
		t.Fatal(err)
	}
	repoRoot := findStringConversionRepoRoot(t)
	t.Setenv("LLGO_ROOT", repoRoot)
	runStringConversionProbe(t, repoRoot, "go", "run", "./cmd/llgo", "run", file)
}
