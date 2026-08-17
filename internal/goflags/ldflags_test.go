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

	"github.com/xgo-dev/llgo/internal/build"
)

func TestParseLinkFlagsOrder(t *testing.T) {
	opts, err := ParseLinkFlags([]string{
		"-tags=integration",
		"-ldflags=-s=false -w=true",
		"-ldflags=-s -w=false",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Present || !opts.Options.OmitSymbolTable {
		t.Fatalf("options = %+v, want present OmitSymbolTable", opts)
	}
	if opts.Options.DWARF != build.DWARFPreserve {
		t.Fatalf("DWARF = %v, want enabled", opts.Options.DWARF)
	}
	if opts.Options.EffectiveOmitDWARF() {
		t.Fatal("EffectiveOmitDWARF() = true, want false")
	}
}

func TestParseLinkFlagsLastArgumentListWins(t *testing.T) {
	opts, err := ParseLinkFlags([]string{
		"-ldflags=-s",
		"-ldflags=-w=false",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Options.OmitSymbolTable {
		t.Fatal("OmitSymbolTable = true, want the final -ldflags list to replace -s")
	}
	if opts.Options.DWARF != build.DWARFPreserve || opts.Options.EffectiveOmitDWARF() {
		t.Fatalf("options = %+v, want explicitly enabled DWARF", opts.Options)
	}
}

func TestParseLinkFlagsExplicitFalse(t *testing.T) {
	opts, err := ParseLinkFlags([]string{"-ldflags=-s=1 -s=f -w=TRUE -w=0"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Options.OmitSymbolTable || opts.Options.DWARF != build.DWARFPreserve || opts.Options.EffectiveOmitDWARF() {
		t.Fatalf("options = %+v, want no symbol stripping and enabled DWARF", opts.Options)
	}
}

func TestParseLinkFlagsDoubleDash(t *testing.T) {
	opts, err := ParseLinkFlags([]string{"-ldflags=--s --w=false"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Options.OmitSymbolTable || opts.Options.EffectiveOmitDWARF() {
		t.Fatalf("options = %+v, want --s with explicit --w=false", opts.Options)
	}
}

func TestParseLinkFlagsSWithExplicitWFalse(t *testing.T) {
	for _, value := range []string{"-s -w=0", "-w=0 -s"} {
		opts, err := ParseLinkFlags([]string{"-ldflags=" + value})
		if err != nil {
			t.Fatal(err)
		}
		if !opts.Options.OmitSymbolTable || opts.Options.EffectiveOmitDWARF() {
			t.Fatalf("%q: options = %+v, want -s with enabled DWARF", value, opts.Options)
		}
	}

	opts, err := ParseLinkFlags([]string{"-ldflags=-s"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Options.EffectiveOmitDWARF() {
		t.Fatal("EffectiveOmitDWARF() = false, want -s to imply -w")
	}
}

func TestParseLinkFlagsQuotesPatternAndIgnored(t *testing.T) {
	opts, err := ParseLinkFlags([]string{
		`-ldflags=all=-s '-w=false' "-extldflags=-static -pthread" '-w=t' -unknown`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Options.OmitSymbolTable || !opts.Options.EffectiveOmitDWARF() {
		t.Fatalf("options = %+v, want -s and effective -w", opts.Options)
	}
	wantIgnored := []string{"-extldflags=-static -pthread", "-unknown"}
	if !reflect.DeepEqual(opts.Ignored, wantIgnored) {
		t.Fatalf("ignored = %q, want %q", opts.Ignored, wantIgnored)
	}
}

func TestParseLinkFlagsBoolSpellings(t *testing.T) {
	trueValues := []string{"1", "t", "T", "true", "TRUE", "True"}
	for _, value := range trueValues {
		t.Run("true_"+value, func(t *testing.T) {
			opts, err := ParseLinkFlags([]string{"-ldflags=-w=" + value})
			if err != nil {
				t.Fatal(err)
			}
			if !opts.Options.EffectiveOmitDWARF() {
				t.Fatalf("-w=%s parsed as false", value)
			}
		})
	}

	falseValues := []string{"0", "f", "F", "false", "FALSE", "False"}
	for _, value := range falseValues {
		t.Run("false_"+value, func(t *testing.T) {
			opts, err := ParseLinkFlags([]string{"-ldflags=-w=" + value})
			if err != nil {
				t.Fatal(err)
			}
			if opts.Options.EffectiveOmitDWARF() {
				t.Fatalf("-w=%s parsed as true", value)
			}
		})
	}
}

func TestParseLinkFlagsErrors(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
	}{
		{name: "missing ldflags value", flags: []string{"-ldflags"}},
		{name: "missing pattern separator", flags: []string{"-ldflags=all"}},
		{name: "missing pattern", flags: []string{"-ldflags==-s"}},
		{name: "unsupported package pattern", flags: []string{"-ldflags=example.com/other=-s"}},
		{name: "quoted first parameter", flags: []string{`-ldflags='-s' -w`}},
		{name: "missing s boolean", flags: []string{"-ldflags=-s="}},
		{name: "missing w boolean", flags: []string{"-ldflags=-w="}},
		{name: "invalid s boolean", flags: []string{"-ldflags=-s=maybe"}},
		{name: "invalid w boolean", flags: []string{"-ldflags=-w=yes"}},
		{name: "unterminated quote", flags: []string{`-ldflags=-s '-w=false`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseLinkFlags(tt.flags); err == nil {
				t.Fatalf("ParseLinkFlags(%q) succeeded, want error", tt.flags)
			}
		})
	}
}

func TestParseLinkFlagsWhitespaceOnly(t *testing.T) {
	opts, err := ParseLinkFlags([]string{"-ldflags= \t\n"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.Present || opts.Options != (build.LinkOptions{}) || len(opts.Ignored) != 0 {
		t.Fatalf("options = %+v, want an empty present list", opts)
	}
}

func TestBuildFlagMatchersRejectBareDash(t *testing.T) {
	for _, arg := range []string{"-", "--"} {
		if _, _, ok := buildParallelFlag(arg); ok {
			t.Fatalf("buildParallelFlag(%q) unexpectedly matched", arg)
		}
		if _, _, _, ok := argumentListBuildFlag(arg); ok {
			t.Fatalf("argumentListBuildFlag(%q) unexpectedly matched", arg)
		}
	}
}
