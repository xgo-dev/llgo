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

package build

import (
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestPrintCompiledPackage(t *testing.T) {
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() { os.Stderr = oldStderr })

	pkg := &aPackage{Package: &packages.Package{PkgPath: "example.com/rebuilt"}}
	printCompiledPackage(&Config{PrintPackages: true}, pkg)
	pkg.CacheHit = true
	printCompiledPackage(&Config{PrintPackages: true}, pkg)
	pkg.CacheHit = false
	printCompiledPackage(&Config{}, pkg)

	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	if want := "example.com/rebuilt\n"; string(got) != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
