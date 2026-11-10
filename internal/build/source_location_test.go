// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

const locationBindings = `package location
import _ "unsafe"
//go:linkname File llgo.file
var File *int8
//go:linkname Line llgo.line
var Line int32
`

func sourceLocationFixture(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":               "module example.com/sourcelocation\n\ngo 1.27\n",
		"location/location.go": locationBindings,
		"main.go":              source,
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		contents = strings.ReplaceAll(contents, "DEFAULT_FILE", strconv.Quote(filepath.Join(dir, "main.go")))
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSourceLocationNative(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs native executables")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := sourceLocationFixture(t, `package main
import (
 "example.com/sourcelocation/location"
 "unsafe"
)
var defaultFile = location.File
//line /generated/init.xgo:100
var initial = location.Line
func generic[T any](x T) int32 {
//line /generated/generic.xgo:300
 return location.Line
}
func helper() (*int8,int32) {
//line /generated/helper.xgo:400
 return location.File, location.Line
}
func checkFile(p *int8, want string) {
 for i:=0;i<len(want);i++ {if byte(*(*int8)(unsafe.Add(unsafe.Pointer(p),i)))!=want[i] {panic("file mismatch")}}
 if *(*int8)(unsafe.Add(unsafe.Pointer(p),len(want)))!=0 {panic("missing NUL")}
}
func main() {
//line /generated/main.xgo:200
 a,b:=location.Line,location.Line
 if a!=200 || b!=200 || initial!=100 || generic(1)!=300 || generic("")!=300 {panic("line mismatch")}
 f,line:=helper(); checkFile(f,"/generated/helper.xgo"); if line!=400 {panic("wrapper location")}
//line /generated/closure.xgo:500
 closure:=func() int32 {return location.Line}
 if closure()!=500 {panic("closure location")}
 checkFile(defaultFile,DEFAULT_FILE)
//line /generated/file.xgo:600
 p,q:=location.File,location.File
 if p!=q {panic("file not deduplicated")}; checkFile(p,"/generated/file.xgo")
 println("ok")
}
`)
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(level.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel = level
			conf.LTO = lto.Off
			conf.OutFile = filepath.Join(dir, "probe-"+level.String())
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "ok" {
				t.Fatalf("run: %v\n%s", err, out)
			}
		})
	}
}

func TestSourceLocationImportedDiagnostics(t *testing.T) {
	for _, test := range []struct{ name, imports, body, want string }{
		{"write", "", `func main() { location.Line = 1 }`, "cannot assign to compiler variable Line"},
		{"unused address", "", `func main() { _ = &location.Line }`, "cannot take address of compiler variable Line"},
		{"erased address", "", `func main() { _ = *(&location.Line) }`, "cannot take address of compiler variable Line"},
		{"dead write", "", `func main() { if false { location.Line++ } }`, "cannot assign to compiler variable Line"},
		{"generic address", "", `func f[T any]() { _ = &location.File }; func main() {}`, "cannot take address of compiler variable File"},
		{"dot import", ".", `func main() { _ = &Line }`, "cannot take address of compiler variable Line"},
		{"renamed import", "pos", `func main() { pos.Line += 1 }`, "cannot assign to compiler variable Line"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := sourceLocationFixture(t, "package main\nimport "+test.imports+" \"example.com/sourcelocation/location\"\n"+test.body)
			conf := NewDefaultConf(ModeGen)
			output := captureStderr(t)
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			diagnostic := output()
			if err == nil || !strings.Contains(diagnostic, test.want) {
				t.Fatalf("error = %v (%s), want %q", err, diagnostic, test.want)
			}
		})
	}
}

func TestSourceLocationAfterMidway(t *testing.T) {
	dir := sourceLocationFixture(t, `package main
import (
 "example.com/sourcelocation/location"
 "simd"
)
//line /generated/vector.xgo:700
func line(v simd.Uint32s) int32 { return location.Line }
func main() { println(line(simd.BroadcastUint32s(1))) }
`)
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pkg := range pkgs {
		defer pkg.LPkg.Prog.Dispose()
		mod := pkg.LPkg.Module()
		if mod.NamedFunction("main.line").IsNil() {
			continue
		}
		found = true
		for _, name := range []string{"main.line@simd0", "main.line@simd128"} {
			fn := mod.NamedFunction(name)
			if fn.IsNil() || !strings.Contains(fn.String(), "ret i32 700") {
				t.Fatalf("rewritten body %s lost its source position", name)
			}
		}
	}
	if !found {
		t.Fatal("missing main package")
	}
}
