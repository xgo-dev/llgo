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

package goflags

import (
	"reflect"
	"testing"

	"github.com/goplus/llgo/internal/build"
	"github.com/goplus/llgo/internal/optlevel"
)

func TestApplyBuildFlagsNormalizesNativeSpellings(t *testing.T) {
	conf := &build.Config{}
	args := []string{
		"--gcflags", "all=-N -l",
		"--ldflags=-s -w=false",
		"-toolexec", "tool --mode check",
	}
	if err := ApplyBuildFlags(conf, args); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-gcflags=all=-N -l",
		"-ldflags=-s -w=false",
		"-toolexec=tool --mode check",
	}
	if !reflect.DeepEqual(conf.GoBuildFlags, want) {
		t.Fatalf("GoBuildFlags = %#v, want %#v", conf.GoBuildFlags, want)
	}
	if conf.OptLevel != optlevel.O0 {
		t.Fatalf("OptLevel = %v, want O0", conf.OptLevel)
	}
	if !conf.LinkOptions.OmitSymbolTable || conf.LinkOptions.EffectiveOmitDWARF() {
		t.Fatalf("LinkOptions = %+v, want -s with explicit -w=false", conf.LinkOptions)
	}
}

func TestApplyBuildFlagsMissingValueIsAtomic(t *testing.T) {
	for _, arg := range []string{"--ldflags", "-p"} {
		t.Run(arg, func(t *testing.T) {
			conf := &build.Config{GoBuildFlags: []string{"-tags=existing"}}
			want := *conf
			want.GoBuildFlags = append([]string(nil), conf.GoBuildFlags...)
			if err := ApplyBuildFlags(conf, []string{arg}); err == nil {
				t.Fatal("ApplyBuildFlags succeeded, want error")
			}
			if !reflect.DeepEqual(*conf, want) {
				t.Fatalf("configuration changed on error:\n got %+v\nwant %+v", *conf, want)
			}
		})
	}
}

func TestApplyBuildFlagsInvalidLinkValueIsAtomic(t *testing.T) {
	conf := &build.Config{GoBuildFlags: []string{"-tags=existing"}}
	want := *conf
	want.GoBuildFlags = append([]string(nil), conf.GoBuildFlags...)
	if err := ApplyBuildFlags(conf, []string{"-ldflags=-w=invalid"}); err == nil {
		t.Fatal("ApplyBuildFlags succeeded, want error")
	}
	if !reflect.DeepEqual(*conf, want) {
		t.Fatalf("configuration changed on error:\n got %+v\nwant %+v", *conf, want)
	}
}

func TestApplyBuildFlagsParallelism(t *testing.T) {
	conf := &build.Config{}
	if err := ApplyBuildFlags(conf, []string{"-p", "2", "--p=4"}); err != nil {
		t.Fatal(err)
	}
	if conf.BuildParallelism != 4 {
		t.Fatalf("BuildParallelism = %d, want 4", conf.BuildParallelism)
	}
	if want := []string{"-p=2", "-p=4"}; !reflect.DeepEqual(conf.GoBuildFlags, want) {
		t.Fatalf("GoBuildFlags = %#v, want %#v", conf.GoBuildFlags, want)
	}
}

func TestApplyBuildFlagsInvalidParallelismIsAtomic(t *testing.T) {
	for _, value := range []string{"0", "-1", "nope"} {
		t.Run(value, func(t *testing.T) {
			conf := &build.Config{
				GoBuildFlags:     []string{"-tags=existing"},
				BuildParallelism: 3,
			}
			want := *conf
			want.GoBuildFlags = append([]string(nil), conf.GoBuildFlags...)
			if err := ApplyBuildFlags(conf, []string{"-p=" + value}); err == nil {
				t.Fatalf("ApplyBuildFlags(-p=%s) succeeded, want error", value)
			}
			if !reflect.DeepEqual(*conf, want) {
				t.Fatalf("configuration changed on error:\n got %+v\nwant %+v", *conf, want)
			}
		})
	}
}

func TestApplyBuildFlagsFrontendGCFlagSemantics(t *testing.T) {
	tests := []struct {
		name           string
		flags          []string
		wantLevel      optlevel.Level
		wantGo         string
		wantSaturating bool
	}{
		{name: "unpatterned", flags: []string{"-gcflags=-lang=go1.25 -N"}, wantLevel: optlevel.O0, wantGo: "go1.25"},
		{name: "all pattern", flags: []string{"-gcflags=all='-lang=go1.24' '-l'"}, wantLevel: optlevel.O0, wantGo: "go1.24"},
		{name: "unrelated pattern", flags: []string{"-gcflags=example.com/other=-N"}},
		{name: "last applicable list wins", flags: []string{"-gcflags=-N -lang=go1.23", "-gcflags="}},
		{name: "converthash enabled", flags: []string{"-gcflags=-d=converthash=qy"}, wantSaturating: true},
		{name: "converthash enabled in debug list", flags: []string{"-gcflags=-d=other=1,converthash=y"}, wantSaturating: true},
		{name: "unrelated debug flag preserves converthash", flags: []string{"-gcflags=-d=converthash=qy -d=other=1"}, wantSaturating: true},
		{name: "converthash verbose all", flags: []string{"-gcflags=-d=converthash=vy"}, wantSaturating: true},
		{name: "converthash quiet verbose all", flags: []string{"-gcflags=-d=converthash=qvy"}, wantSaturating: true},
		{name: "converthash double negation", flags: []string{"-gcflags=-d=converthash=!!y"}, wantSaturating: true},
		{name: "converthash negated no", flags: []string{"-gcflags=-d=converthash=!n"}, wantSaturating: true},
		{name: "converthash disabled", flags: []string{"-gcflags=-d=converthash=qn"}},
		{name: "converthash uppercase is invalid", flags: []string{"-gcflags=-d=converthash=QY"}},
		{name: "last converthash debug value wins", flags: []string{"-gcflags=-d=converthash=qy,converthash=qn"}},
		{name: "last converthash list wins", flags: []string{"-gcflags=-d=converthash=qy", "-gcflags=-d=converthash=qn"}},
		{name: "package converthash ignored", flags: []string{"-gcflags=example.com/other=-d=converthash=qy"}},
		{name: "N false", flags: []string{"-gcflags=-N=false"}},
		{name: "N zero", flags: []string{"-gcflags=-N=0"}},
		{name: "l false", flags: []string{"-gcflags=-l=false"}},
		{name: "l zero", flags: []string{"-gcflags=-l=0"}},
		{name: "l debug count", flags: []string{"-gcflags=-l=4"}},
		{name: "N true", flags: []string{"-gcflags=-N=true"}, wantLevel: optlevel.O0},
		{name: "l true", flags: []string{"-gcflags=-l=true"}, wantLevel: optlevel.O0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &build.Config{}
			if err := ApplyBuildFlags(conf, tt.flags); err != nil {
				t.Fatal(err)
			}
			if conf.OptLevel != tt.wantLevel || conf.GoVersion != tt.wantGo || conf.SaturatingFloatToUint32 != tt.wantSaturating {
				t.Fatalf("frontend config = (%v, %q, saturating=%v), want (%v, %q, saturating=%v)", conf.OptLevel, conf.GoVersion, conf.SaturatingFloatToUint32, tt.wantLevel, tt.wantGo, tt.wantSaturating)
			}
		})
	}
}

func TestCompilerDebugFlagsLastValueWins(t *testing.T) {
	got := make(compilerDebugFlags)
	got.apply("fmahash=qy,loopvarhash=n,checkptr")
	got.apply("fmahash=qn")
	want := compilerDebugFlags{
		"fmahash":     "qn",
		"loopvarhash": "n",
		"checkptr":    "1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compiler debug flags = %#v, want %#v", got, want)
	}
}

func TestBisectPatternAlwaysEnabled(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{pattern: "y", want: true},
		{pattern: "qy", want: true},
		{pattern: "vy", want: true},
		{pattern: "qvvy", want: true},
		{pattern: "!!y", want: true},
		{pattern: "!n", want: true},
		{pattern: ""},
		{pattern: "n"},
		{pattern: "qn"},
		{pattern: "!y"},
		{pattern: "01"},
		{pattern: "QY"},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := bisectPatternAlwaysEnabled(tt.pattern); got != tt.want {
				t.Fatalf("bisectPatternAlwaysEnabled(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}
