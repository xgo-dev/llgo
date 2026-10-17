/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package flags

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/internal/base"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestExecFlagQuotedWords(t *testing.T) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	words := AddExecFlag(fs)
	if err := fs.Parse([]string{`-exec="path with spaces/wrapper" 'one argument' literal;word`}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"path with spaces/wrapper", "one argument", "literal;word"}; !reflect.DeepEqual(*words, want) {
		t.Fatalf("execution wrapper = %q, want %q", *words, want)
	}
	before := append([]string{}, (*words)...)
	if err := fs.Parse([]string{`-exec="unterminated`}); err == nil {
		t.Fatal("unterminated wrapper accepted")
	}
	if !reflect.DeepEqual(*words, before) {
		t.Fatal("malformed wrapper changed the execution command")
	}
}

func TestMemProfileRateFullCountingOnly(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	AddTestBinaryFlags(fs)
	if err := fs.Parse([]string{"-test.memprofilerate=1"}); err != nil || TestMemProfileRate != 1 {
		t.Fatalf("full-counting mode: rate=%d error=%v", TestMemProfileRate, err)
	}
	for _, value := range []string{"0", "2", "512000", "invalid"} {
		if err := fs.Parse([]string{"-memprofilerate=" + value}); err == nil || TestMemProfileRate != 1 {
			t.Fatalf("unsupported rate %q changed the mode: rate=%d error=%v", value, TestMemProfileRate, err)
		}
	}
}

func TestOtherCommandsIgnoreKnownTestAliasDefaults(t *testing.T) {
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "-test.run=TestAlias")
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	AddCommonFlags(fs)
	if err := Parse(fs, []string{"."}); err != nil {
		t.Fatalf("build rejected a known test-only GOFLAGS default: %v", err)
	}
	t.Setenv("GOFLAGS", "-test.json")
	if err := Parse(fs, []string{"."}); err == nil {
		t.Fatal("accepted an unknown test-binary alias in GOFLAGS")
	}
}

func TestCompilerOptionSpellingsFromCLIAndGOFLAGS(t *testing.T) {
	t.Setenv("GOENV", "off")
	for _, tc := range []struct {
		value          string
		wantGo         string
		wantNoOpt      bool
		wantSaturating bool
	}{
		{value: "-lang=go1.21", wantGo: "go1.21"},
		{value: "-lang go1.21", wantGo: "go1.21"},
		{value: "--lang go1.21", wantGo: "go1.21"},
		{value: "-d converthash=qy", wantSaturating: true},
		{value: "--d=converthash=qy", wantSaturating: true},
		{value: "--N", wantNoOpt: true},
		{value: "--l", wantNoOpt: true},
		{value: "-N=-1 -N"},
	} {
		for _, defaults := range []bool{false, true} {
			source := "CLI"
			if defaults {
				source = "GOFLAGS"
			}
			t.Run(source+"/"+tc.value, func(t *testing.T) {
				args := []string{"-gcflags=" + tc.value, "."}
				value := ""
				if defaults {
					value = strconv.Quote(args[0])
					args = args[1:]
				}
				t.Setenv("GOFLAGS", value)
				cmd := &base.Command{}
				cmd.Flag.Init("build", flag.ContinueOnError)
				captured := CaptureGoBuildFlags(cmd)
				if err := Parse(&cmd.Flag, args); err != nil {
					t.Fatal(err)
				}
				conf := &build.Config{OptLevel: optlevel.O2}
				if err := ApplyGoBuildFlags(conf, captured.Args); err != nil {
					t.Fatal(err)
				}
				wantLevel := optlevel.O2
				if tc.wantNoOpt {
					wantLevel = optlevel.O0
				}
				if conf.GoVersion != tc.wantGo || conf.OptLevel != wantLevel || conf.SaturatingFloatToInt != tc.wantSaturating {
					t.Fatalf("compiler options were not applied: version=%q level=%v saturating=%v", conf.GoVersion, conf.OptLevel, conf.SaturatingFloatToInt)
				}
				if os.Getenv("GOFLAGS") != value {
					t.Fatal("changed the caller's GOFLAGS")
				}
			})
		}
	}
}

func TestGOFLAGSCompilerListOverride(t *testing.T) {
	t.Setenv("GOENV", "off")
	for _, tc := range []struct {
		defaults  string
		explicit  string
		wantError bool
	}{
		{defaults: "-gcflags=-m", explicit: "-gcflags=-N"},
		{defaults: "-gcflags=all=-m", explicit: "-gcflags=all=-N"},
		{defaults: "-gcflags=all=-m", explicit: "-gcflags=-N", wantError: true},
	} {
		t.Run(tc.defaults+"/"+tc.explicit, func(t *testing.T) {
			t.Setenv("GOFLAGS", tc.defaults)
			cmd := &base.Command{}
			cmd.Flag.Init("build", flag.ContinueOnError)
			captured := CaptureGoBuildFlags(cmd)
			if err := Parse(&cmd.Flag, []string{tc.explicit, "."}); err != nil {
				t.Fatal(err)
			}
			conf := &build.Config{OptLevel: optlevel.O2}
			err := ApplyGoBuildFlags(conf, captured.Args)
			if tc.wantError {
				if err == nil {
					t.Fatal("accepted compiler options still active for dependencies")
				}
			} else if err != nil || conf.OptLevel != optlevel.O0 {
				t.Fatalf("explicit compiler list did not replace defaults: level=%v err=%v", conf.OptLevel, err)
			}
			if os.Getenv("GOFLAGS") != tc.defaults {
				t.Fatal("changed the caller's GOFLAGS")
			}
		})
	}
}

func TestDirectoryAndGOFLAGSPrecedence(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Error(err)
		}
	})
	for _, args := range [][]string{{"-C", dir, "-tags=explicit", "."}, {"-C=" + dir, "-tags=explicit", "."}, {"--C", dir, "-tags=explicit", "."}} {
		t.Setenv("GOENV", "off")
		t.Setenv("GOFLAGS", "-tags=env -v")
		var tags string
		var verbose bool
		fs := flag.NewFlagSet("native", flag.ContinueOnError)
		AddDirectoryFlag(fs)
		fs.StringVar(&tags, "tags", "", "tags")
		fs.BoolVar(&verbose, "v", false, "verbose")
		if err := Parse(fs, args); err != nil {
			t.Fatal(err)
		}
		cwd, _ := os.Getwd()
		if !sameDirectory(cwd, dir) || tags != "explicit" || !verbose || !reflect.DeepEqual(fs.Args(), []string{"."}) {
			t.Fatalf("cwd=%s tags=%s verbose=%v operands=%v", cwd, tags, verbose, fs.Args())
		}
	}
}

func TestInvalidDefaultsAndDirectory(t *testing.T) {
	t.Setenv("GOENV", "off")
	for _, value := range []string{"bare", "-", "--", "---v", "-=x", "--=x", "-tags", "-C=tmp", "'-tags=unterminated", "-v=invalid", "-unknown"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GOFLAGS", value)
			fs := flag.NewFlagSet("invalid", flag.ContinueOnError)
			fs.SetOutput(new(bytes.Buffer))
			AddCommonFlags(fs)
			fs.String("tags", "", "tags")
			if err := Parse(fs, nil); err == nil {
				t.Fatalf("accepted GOFLAGS=%q", value)
			}
		})
	}
	t.Setenv("GOFLAGS", "")
	for _, args := range [][]string{{"-C"}, {"-C="}, {"-C", filepath.Join(t.TempDir(), "missing")}, {"-v", "-C", t.TempDir()}} {
		fs := flag.NewFlagSet("invalid", flag.ContinueOnError)
		fs.SetOutput(new(bytes.Buffer))
		AddCommonFlags(fs)
		if err := Parse(fs, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	fs := flag.NewFlagSet("valid", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	AddCommonFlags(fs)
	if err := Parse(fs, []string{"operand", "-C=program-argument"}); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultsApplyOnlyToRelevantCommands(t *testing.T) {
	t.Setenv("GOENV", "off")
	t.Setenv("GOFLAGS", "-run=Selected -port=device -tags=default")
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	tags := fs.String("tags", "", "tags")
	if err := Parse(fs, nil); err != nil || *tags != "default" {
		t.Fatalf("irrelevant defaults affected build: tags=%q err=%v", *tags, err)
	}
}

func TestUnsupportedBuildFlagsHaveHelpAndFail(t *testing.T) {
	for _, arg := range []string{"-race", "-race=false", "-msan", "-asan", "-pgo=auto", "-asmflags=-S", "-compiler=gc", "-gccgoflags=-O2", "-installsuffix=other", "-pkgdir=tmp", "-toolexec=wrapper", "-buildvcs=false", "-linkshared", "-n", "-work", "-trimpath"} {
		t.Run(arg, func(t *testing.T) {
			cmd := &base.Command{}
			CaptureGoBuildFlags(cmd)
			cmd.Flag.SetOutput(new(bytes.Buffer))
			if err := cmd.Flag.Parse([]string{arg}); err == nil || !strings.Contains(err.Error(), "unsupported") {
				t.Fatalf("accepted %s: %v", arg, err)
			}
		})
	}
	for _, arg := range []string{"-modcacherw=invalid"} {
		cmd := &base.Command{}
		CaptureGoBuildFlags(cmd)
		cmd.Flag.SetOutput(new(bytes.Buffer))
		if err := cmd.Flag.Parse([]string{arg}); err == nil {
			t.Fatalf("accepted %s", arg)
		}
	}
	fs := flag.NewFlagSet("buildenv", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	AddBuildFlags(fs)
	if fs.Lookup("buildenv") != nil {
		t.Fatal("dead buildenv flag still registered")
	}
}

func sameDirectory(a, b string) bool {
	x, e := os.Stat(a)
	if e != nil {
		return false
	}
	y, e := os.Stat(b)
	return e == nil && os.SameFile(x, y)
}

func TestPersistentDefaultsAndLookupErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "goenv")
	if err := os.WriteFile(file, []byte("GOFLAGS=-tags=persisted\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", file)
	t.Setenv("GOFLAGS", "")
	fs := flag.NewFlagSet("persistent", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer))
	tags := fs.String("tags", "", "tags")
	if err := Parse(fs, nil); err != nil || *tags != "persisted" {
		t.Fatalf("defaults=%q err=%v", *tags, err)
	}
	t.Setenv("PATH", "")
	if err := Parse(fs, nil); err == nil {
		t.Fatal("accepted missing Go toolchain")
	}
}

func TestManifestRejectionsAreRecognized(t *testing.T) {
	for _, command := range []string{"build", "run", "install", "test", "clean", "version"} {
		fs := flag.NewFlagSet(command, flag.ContinueOnError)
		fs.SetOutput(new(bytes.Buffer))
		FinalizeNativeFlags(command, fs)
		fs.VisitAll(func(f *flag.Flag) {
			if err := fs.Set(f.Name, "true"); err == nil || !strings.Contains(err.Error(), "unsupported") {
				t.Errorf("%s -%s did not reject: %v", command, f.Name, err)
			}
		})
	}
}
