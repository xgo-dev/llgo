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
	"reflect"
	"strings"
	"testing"
)

func TestResolveGOARCHConfig(t *testing.T) {
	tests := []struct {
		name string
		conf Config
		env  map[string]string
		want Config
	}{
		{
			name: "386 default",
			conf: Config{Goarch: "386"},
			want: Config{Goarch: "386", GO386: "sse2"},
		},
		{
			name: "386 environment",
			conf: Config{Goarch: "386", GOAMD64: "v4"},
			env:  map[string]string{"GO386": "softfloat"},
			want: Config{Goarch: "386", GO386: "softfloat"},
		},
		{
			name: "386 explicit wins",
			conf: Config{Goarch: "386", GO386: "softfloat"},
			env:  map[string]string{"GO386": "sse2"},
			want: Config{Goarch: "386", GO386: "softfloat"},
		},
		{
			name: "amd64 explicit wins",
			conf: Config{Goarch: "amd64", GOAMD64: "v3"},
			env:  map[string]string{"GOAMD64": "v4"},
			want: Config{Goarch: "amd64", GOAMD64: "v3"},
		},
		{
			name: "amd64 environment",
			conf: Config{Goarch: "amd64"},
			env:  map[string]string{"GOAMD64": "v4"},
			want: Config{Goarch: "amd64", GOAMD64: "v4"},
		},
		{
			name: "arm default",
			conf: Config{Goarch: "arm"},
			want: Config{Goarch: "arm", GOARM: "7,hardfloat"},
		},
		{
			name: "arm environment",
			conf: Config{Goarch: "arm"},
			env:  map[string]string{"GOARM": "6,softfloat"},
			want: Config{Goarch: "arm", GOARM: "6,softfloat"},
		},
		{
			name: "arm5 canonical default float mode",
			conf: Config{Goarch: "arm", GOARM: "5"},
			want: Config{Goarch: "arm", GOARM: "5,softfloat"},
		},
		{
			name: "arm6 canonical default float mode",
			conf: Config{Goarch: "arm", GOARM: "6"},
			want: Config{Goarch: "arm", GOARM: "6,hardfloat"},
		},
		{
			name: "arm explicit wins",
			conf: Config{Goarch: "arm", GOARM: "5,hardfloat"},
			env:  map[string]string{"GOARM": "7"},
			want: Config{Goarch: "arm", GOARM: "5,hardfloat"},
		},
		{
			name: "arm64 canonical extensions",
			conf: Config{Goarch: "arm64", GOARM64: "v8.2,crypto"},
			want: Config{Goarch: "arm64", GOARM64: "v8.2,lse,crypto"},
		},
		{
			name: "arm64 environment",
			conf: Config{Goarch: "arm64"},
			env:  map[string]string{"GOARM64": "v9.1"},
			want: Config{Goarch: "arm64", GOARM64: "v9.1,lse"},
		},
		{
			name: "unrelated architecture",
			conf: Config{Goarch: "wasm", GO386: "softfloat", GOAMD64: "v4", GOARM: "5", GOARM64: "v9.5"},
			want: Config{Goarch: "wasm"},
		},
		{
			name: "named target ignores architecture environment",
			conf: Config{Goarch: "amd64", GOAMD64: "v4", GOARM: "5", Target: "wasi"},
			want: Config{Goarch: "amd64", Target: "wasi"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getenv := func(name string) string { return test.env[name] }
			if err := resolveGOARCHConfig(&test.conf, getenv); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(test.conf, test.want) {
				t.Fatalf("config = %+v, want %+v", test.conf, test.want)
			}
		})
	}
}

func TestResolveBuildConfigInvalidGOARCHValueIsAtomic(t *testing.T) {
	for _, input := range []*Config{
		{Goarch: "amd64", GOAMD64: "v5", GoBuildFlags: []string{"-tags=keep"}},
		{Goarch: "arm", GOARM: "8", GoBuildFlags: []string{"-tags=keep"}},
	} {
		want := input.clone()
		if _, err := resolveBuildConfig(input); err == nil {
			t.Fatalf("resolveBuildConfig succeeded with %+v", input)
		}
		if !reflect.DeepEqual(input, want) {
			t.Fatalf("input changed on error:\n got %+v\nwant %+v", input, want)
		}
	}
}

func TestGOARCHEnv(t *testing.T) {
	tests := []struct {
		conf *Config
		want []string
	}{
		{},
		{conf: &Config{Goarch: "386", GO386: "softfloat"}, want: []string{"GO386=softfloat", "GOAMD64=", "GOARM=", "GOARM64="}},
		{conf: &Config{Goarch: "amd64", GOAMD64: "v4"}, want: []string{"GO386=", "GOAMD64=v4", "GOARM=", "GOARM64="}},
		{conf: &Config{Goarch: "arm", GOARM: "6,softfloat"}, want: []string{"GO386=", "GOAMD64=", "GOARM=6,softfloat", "GOARM64="}},
		{conf: &Config{Goarch: "arm64", GOARM64: "v9.5,lse,crypto"}, want: []string{"GO386=", "GOAMD64=", "GOARM=", "GOARM64=v9.5,lse,crypto"}},
		{conf: &Config{Goarch: "amd64", Target: "wasi"}, want: []string{"GO386=", "GOAMD64=", "GOARM=", "GOARM64="}},
	}
	for _, test := range tests {
		if got := goarchEnv(test.conf); !reflect.DeepEqual(got, test.want) {
			t.Errorf("goarchEnv(%+v) = %q, want %q", test.conf, got, test.want)
		}
	}
}

func TestGOARCHConfigSeparatesCacheFingerprints(t *testing.T) {
	manifest := func(conf *Config) (string, string) {
		m := newManifestBuilder()
		ctx := &context{buildConf: conf, llvmVersion: "test"}
		ctx.collectEnvInputs(m)
		return m.Build(), m.Fingerprint()
	}
	for _, test := range []struct {
		name       string
		left       *Config
		right      *Config
		leftEntry  string
		rightEntry string
	}{
		{
			name:       "GO386",
			left:       &Config{Goos: "windows", Goarch: "386", GO386: "sse2"},
			right:      &Config{Goos: "windows", Goarch: "386", GO386: "softfloat"},
			leftEntry:  "GO386: sse2",
			rightEntry: "GO386: softfloat",
		},
		{
			name:       "GOAMD64",
			left:       &Config{Goos: "windows", Goarch: "amd64", GOAMD64: "v1"},
			right:      &Config{Goos: "windows", Goarch: "amd64", GOAMD64: "v4"},
			leftEntry:  "GOAMD64: v1",
			rightEntry: "GOAMD64: v4",
		},
		{
			name:       "GOARM",
			left:       &Config{Goos: "linux", Goarch: "arm", GOARM: "7,hardfloat"},
			right:      &Config{Goos: "linux", Goarch: "arm", GOARM: "6,softfloat"},
			leftEntry:  "GOARM: 7,hardfloat",
			rightEntry: "GOARM: 6,softfloat",
		},
		{
			name:       "GOARM64",
			left:       &Config{Goos: "windows", Goarch: "arm64", GOARM64: "v8.0"},
			right:      &Config{Goos: "windows", Goarch: "arm64", GOARM64: "v9.5,lse,crypto"},
			leftEntry:  "GOARM64: v8.0",
			rightEntry: "GOARM64: v9.5,lse,crypto",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			leftManifest, leftFingerprint := manifest(test.left)
			rightManifest, rightFingerprint := manifest(test.right)
			if leftFingerprint == rightFingerprint {
				t.Fatalf("%s configurations share a cache fingerprint", test.name)
			}
			if !strings.Contains(leftManifest, test.leftEntry) || !strings.Contains(rightManifest, test.rightEntry) {
				t.Fatalf("architecture settings missing from manifests:\n%s\n%s", leftManifest, rightManifest)
			}
		})
	}
}
