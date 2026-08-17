//go:build llgo

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

package localitymulti

import (
	"testing"

	localityblock0 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p0"
	localityblock1 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p1"
	localityblock2 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p2"
	localityblock3 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p3"
	localityblock4 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p4"
	localityblock5 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p5"
	localityblock6 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p6"
	localityblock7 "github.com/xgo-dev/llgo/test/llgoext/testdata/localityblocks/p7"
)

var benchmarkSink uintptr

func TestGLSPackageWorkingSet(t *testing.T) {
	prepare := []func(){
		localityblock0.Prepare,
		localityblock1.Prepare,
		localityblock2.Prepare,
		localityblock3.Prepare,
		localityblock4.Prepare,
		localityblock5.Prepare,
		localityblock6.Prepare,
		localityblock7.Prepare,
	}
	read := []func() uintptr{
		localityblock0.Read,
		localityblock1.Read,
		localityblock2.Read,
		localityblock3.Read,
		localityblock4.Read,
		localityblock5.Read,
		localityblock6.Read,
		localityblock7.Read,
	}
	for i := range prepare {
		prepare[i]()
		if got := read[i](); got == 0 {
			t.Fatalf("package %d GLS pointer is nil", i)
		}
	}
}

func BenchmarkGLSPackageWorkingSet2(b *testing.B) {
	localityblock0.Prepare()
	localityblock1.Prepare()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localityblock0.Read()
		value += localityblock1.Read()
	}
	benchmarkSink = value
}

func BenchmarkGLSPackageWorkingSet4(b *testing.B) {
	localityblock0.Prepare()
	localityblock1.Prepare()
	localityblock2.Prepare()
	localityblock3.Prepare()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localityblock0.Read()
		value += localityblock1.Read()
		value += localityblock2.Read()
		value += localityblock3.Read()
	}
	benchmarkSink = value
}

func BenchmarkGLSPackageWorkingSet8(b *testing.B) {
	localityblock0.Prepare()
	localityblock1.Prepare()
	localityblock2.Prepare()
	localityblock3.Prepare()
	localityblock4.Prepare()
	localityblock5.Prepare()
	localityblock6.Prepare()
	localityblock7.Prepare()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localityblock0.Read()
		value += localityblock1.Read()
		value += localityblock2.Read()
		value += localityblock3.Read()
		value += localityblock4.Read()
		value += localityblock5.Read()
		value += localityblock6.Read()
		value += localityblock7.Read()
	}
	benchmarkSink = value
}
