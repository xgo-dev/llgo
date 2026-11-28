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
	"github.com/xgo-dev/llgo/types"
	"os"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/packages"
)

func captureStderr(t *testing.T) func() string {
	t.Helper()
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	closed := false
	t.Cleanup(func() {
		if !closed {
			os.Stderr = oldStderr
			_ = stderr.Close()
		}
	})
	return func() string {
		t.Helper()
		if closed {
			t.Fatal("stderr capture already read")
		}
		os.Stderr = oldStderr
		closed = true
		if err := stderr.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(stderr.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(got)
	}
}

func TestFinalizePackageBuildPrintsCompletedPackage(t *testing.T) {
	readStderr := captureStderr(t)

	pkg := &aPackage{Package: &packages.Package{
		PkgPath: "example.com/rebuilt",
		Types:   types.NewPackage("example.com/rebuilt", "rebuilt"),
	}}
	ctx := &context{buildConf: &Config{PrintPackages: true}}
	if err := finalizePackageBuild(ctx, newPackageBuildTask(pkg), false); err != nil {
		t.Fatal(err)
	}

	pkg.CacheHit = true
	if err := finalizePackageBuild(ctx, newPackageBuildTask(pkg), false); err != nil {
		t.Fatal(err)
	}

	pkg.CacheHit = false
	if err := finalizePackageBuild(&context{buildConf: &Config{}}, newPackageBuildTask(pkg), false); err != nil {
		t.Fatal(err)
	}

	if got, want := readStderr(), "example.com/rebuilt\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
