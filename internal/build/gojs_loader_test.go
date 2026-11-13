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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestNeedsGoJSLoader(t *testing.T) {
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm"}
	gojs := &crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}
	if !needsGoJSLoader(conf, gojs, "app.mjs") {
		t.Fatal("raw GoJS build did not publish wasm_exec.js")
	}
	for _, test := range []struct {
		name   string
		conf   *Config
		target *crosscompile.Export
		output string
	}{
		{"Emscripten target", conf, &crosscompile.Export{WasmProvider: crosscompile.WasmProviderEmscripten}, "app.mjs"},
		{"run", &Config{Mode: ModeRun, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm"}, gojs, "app.mjs"},
		{"archive", &Config{Mode: ModeBuild, BuildMode: BuildModeCArchive, Goos: "js", Goarch: "wasm"}, gojs, "app.mjs"},
		{"module", conf, gojs, "app.wasm"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if needsGoJSLoader(test.conf, test.target, test.output) {
				t.Fatal("unexpected GoJS loader")
			}
		})
	}
}

func TestCombineGoJSExec(t *testing.T) {
	got := combineGoJSExec("app.wasm", []byte("factory"), []byte("loader"))
	want := goJSWasmMarkerPrefix + "app.wasm\nfactory\n" + string(goJSRegisterFactory("app.wasm")) + "loader\n"
	if string(got) != want {
		t.Fatalf("combined = %q, want %q", got, want)
	}
	if goJSWasmMarker(got) != "app.wasm" {
		t.Fatalf("marker = %q", goJSWasmMarker(got))
	}
}

func TestPublishGoJSLoaderFrom(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "targets", goJSLoaderScriptName)
	if err := os.WriteFile(src, []byte("loader"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "app.wasm")
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: output}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	driver := emscriptenDriverOutput(conf, output)
	for path, content := range map[string]string{output: "module", driver: "implementation", goJSLoaderSidecar(driver): "stale loader"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if i > 0 {
			if err := os.WriteFile(driver, []byte("implementation"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := publishGoJSLoaderFrom(ctx, driver, false, root); err != nil {
			t.Fatal(err)
		}
		combined := string(combineGoJSExec("app.wasm", []byte("implementation"), []byte("loader")))
		got, err := os.ReadFile(goJSLoaderSidecar(driver))
		if err != nil || string(got) != combined {
			t.Fatalf("wasm_exec.js = %q, %v", got, err)
		}
		if _, err := os.Stat(goJSUniqueLoaderSidecar(driver)); !os.IsNotExist(err) {
			t.Fatalf("unique host created for single program: %v", err)
		}
		if _, err := os.Stat(driver); !os.IsNotExist(err) {
			t.Fatalf("intermediate factory remains: %v", err)
		}
		if got, err := os.ReadFile(output); err != nil || string(got) != "module" {
			t.Fatalf("requested output = %q, %v", got, err)
		}
	}
}

func TestPublishGoJSLoaderSharedAliasByWasmName(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "targets", goJSLoaderScriptName)
	if err := os.WriteFile(src, []byte("loader"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first := filepath.Join(dir, "first.wasm")
	second := filepath.Join(dir, "second.wasm")
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: first}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	firstDriver := emscriptenDriverOutput(conf, first)
	if err := os.WriteFile(firstDriver, []byte("first-glue"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, firstDriver, false, root); err != nil {
		t.Fatal(err)
	}
	wantFirst := string(combineGoJSExec("first.wasm", []byte("first-glue"), []byte("loader")))
	got, err := os.ReadFile(goJSLoaderSidecar(firstDriver))
	if err != nil || string(got) != wantFirst {
		t.Fatalf("shared after first = %q, %v", got, err)
	}
	if _, err := os.Stat(goJSUniqueLoaderSidecar(firstDriver)); !os.IsNotExist(err) {
		t.Fatalf("unique glue after first: %v", err)
	}
	conf.OutFile = first
	if err := os.WriteFile(firstDriver, []byte("first-glue-v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, firstDriver, false, root); err != nil {
		t.Fatal(err)
	}
	wantFirstV2 := string(combineGoJSExec("first.wasm", []byte("first-glue-v2"), []byte("loader")))
	got, err = os.ReadFile(goJSLoaderSidecar(firstDriver))
	if err != nil || string(got) != wantFirstV2 {
		t.Fatalf("shared after rebuild = %q, %v", got, err)
	}
	conf.OutFile = second
	secondDriver := emscriptenDriverOutput(conf, second)
	if err := os.WriteFile(secondDriver, []byte("second-glue"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, secondDriver, false, root); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(goJSLoaderSidecar(secondDriver))
	if err != nil || string(got) != wantFirstV2 {
		t.Fatalf("shared after second = %q, %v; want first's glue", got, err)
	}
	got, err = os.ReadFile(goJSUniqueLoaderSidecar(secondDriver))
	if err != nil || string(got) != string(combineGoJSExec("second.wasm", []byte("second-glue"), []byte("loader"))) {
		t.Fatalf("second unique glue = %q, %v", got, err)
	}

	dir2 := t.TempDir()
	legacy := filepath.Join(dir2, "app.wasm")
	conf.OutFile = legacy
	legacyDriver := emscriptenDriverOutput(conf, legacy)
	if err := os.WriteFile(goJSLoaderSidecar(legacyDriver), []byte("legacy-unmarked"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyDriver, []byte("app-glue"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, legacyDriver, false, root); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(goJSLoaderSidecar(legacyDriver))
	if err != nil || string(got) != string(combineGoJSExec("app.wasm", []byte("app-glue"), []byte("loader"))) {
		t.Fatalf("unmarked shared after rebuild = %q, %v", got, err)
	}
	if _, err := os.Stat(goJSUniqueLoaderSidecar(legacyDriver)); !os.IsNotExist(err) {
		t.Fatalf("unique glue for unmarked rebuild: %v", err)
	}
}

func TestPublishGoJSLoaderCollisions(t *testing.T) {
	dir := t.TempDir()
	driver := filepath.Join(dir, "app.mjs")
	output := filepath.Join(dir, "app.wasm")
	if err := os.WriteFile(driver, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(driver, goJSLoaderSidecar(driver)); err != nil {
		t.Fatal(err)
	}
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: output}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	err := publishGoJSLoaderFrom(ctx, driver, false, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("collision error = %v", err)
	}
	if got, err := os.ReadFile(driver); err != nil || string(got) != "original" {
		t.Fatalf("collision changed driver: %q, %v", got, err)
	}
}

func TestPublishGoJSLoaderErrorsAndSkips(t *testing.T) {
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm"}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	driver := filepath.Join(t.TempDir(), "app.mjs")
	if err := os.WriteFile(driver, []byte("driver"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"", t.TempDir()} {
		if err := publishGoJSLoaderFrom(ctx, driver, false, root); err == nil {
			t.Fatal("missing root/loader succeeded")
		}
		if got, err := os.ReadFile(driver); err != nil || string(got) != "driver" {
			t.Fatalf("driver output changed: %q, %v", got, err)
		}
	}
	for _, ctx := range []*context{nil, {}, {buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderEmscripten}}} {
		if err := publishGoJSLoaderFrom(ctx, driver, false, ""); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishGoJSLoaderFailurePaths(t *testing.T) {
	for _, name := range []string{"source collision", "source directory", "missing driver", "loader directory", "invalid path"} {
		t.Run(name, func(t *testing.T) {
			root, dir := t.TempDir(), t.TempDir()
			src := filepath.Join(root, "targets", goJSLoaderScriptName)
			if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(src, []byte("loader"), 0o644); err != nil {
				t.Fatal(err)
			}
			driver := filepath.Join(dir, "app.mjs")
			if err := os.WriteFile(driver, []byte("driver"), 0o644); err != nil {
				t.Fatal(err)
			}
			want := ""
			switch name {
			case "source collision":
				if err := os.Link(src, goJSUniqueLoaderSidecar(driver)); err != nil {
					t.Fatal(err)
				}
				want = "collides"
			case "source directory":
				if err := os.Remove(src); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(src, 0o755); err != nil {
					t.Fatal(err)
				}
				want = "source is not a regular file"
			case "missing driver":
				if err := os.Remove(driver); err != nil {
					t.Fatal(err)
				}
				want = "copy " + goJSLoaderScriptName
			case "loader directory":
				path := goJSLoaderSidecar(driver)
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
				want = "copy " + filepath.Base(path)
			case "invalid path":
				driver = filepath.Join(dir, "invalid\x00.mjs")
				want = "publish GoJS host"
			}
			conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm"}
			ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
			if err := publishGoJSLoaderFrom(ctx, driver, false, root); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
			if name != "missing driver" && name != "invalid path" {
				if data, err := os.ReadFile(driver); err != nil || string(data) != "driver" {
					t.Fatalf("driver changed on failure: %q, %v", data, err)
				}
			}
			if name == "source directory" {
				if _, err := os.Stat(goJSUniqueLoaderSidecar(driver)); !os.IsNotExist(err) {
					t.Fatalf("invalid loader source published unique loader: %v", err)
				}
			}
		})
	}
}

func TestExecuteMainLinkPublishesGoJSLoader(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing-loader=%v", missing), func(t *testing.T) {
			t.Setenv("LLGO_TEST_LINKER_HELPER", "write-emscripten")
			root := writeFakeLLGoRoot(t, false)
			t.Setenv("LLGO_ROOT", root)
			if err := os.Mkdir(filepath.Join(root, "targets"), 0o755); err != nil {
				t.Fatal(err)
			}
			if !missing {
				if err := os.WriteFile(filepath.Join(root, "targets", goJSLoaderScriptName), []byte("loader"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			output := filepath.Join(t.TempDir(), "app.wasm")
			conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", PCLNMode: PCLNNone}
			ctx := &context{mode: ModeBuild, buildConf: conf, crossCompile: crosscompile.Export{CC: os.Args[0], WasmProvider: crosscompile.WasmProviderGoJS}}
			err := executeMainLink(ctx, &mainLinkPlan{outputPath: output}, true)
			if missing {
				if err == nil || !strings.Contains(err.Error(), "copy "+goJSLoaderScriptName) {
					t.Fatalf("missing loader error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			combined := string(combineGoJSExec("app.wasm", []byte("export default async function () {}"), []byte("loader")))
			for file, want := range map[string]string{
				output:                    "\x00asm\x01\x00\x00\x00",
				goJSLoaderSidecar(output): combined,
			} {
				if got, err := os.ReadFile(file); err != nil || string(got) != want {
					t.Fatalf("%s = %q, %v; want %q", file, got, err, want)
				}
			}
			if _, err := os.Stat(goJSUniqueLoaderSidecar(output)); !os.IsNotExist(err) {
				t.Fatalf("unique glue created for single program: %v", err)
			}
		})
	}
}

func TestGoJSLoaderLinkFlags(t *testing.T) {
	for _, mode := range []Mode{ModeBuild, ModeRun, ModeTest} {
		conf := &Config{Mode: mode, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm"}
		for _, provider := range []crosscompile.WasmProvider{crosscompile.WasmProviderGoJS, crosscompile.WasmProviderEmscripten} {
			target := &crosscompile.Export{WasmProvider: provider}
			flags := goJSLoaderLinkFlags(conf, target)
			if got, want := len(flags) != 0, mode == ModeBuild && provider == crosscompile.WasmProviderGoJS; got != want {
				t.Fatalf("mode=%v provider=%v flags=%v", mode, provider, flags)
			}
		}
	}
	for _, output := range []string{"app.mjs", "app.js", "app.html"} {
		conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: output}
		if flags := goJSLoaderLinkFlags(conf, &crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}); len(flags) != 0 {
			t.Fatalf("explicit %s changed the module factory: %v", output, flags)
		}
	}
}

func TestGoJSLoaderJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required")
	}
	loader, err := filepath.Abs(filepath.Join("..", "..", "targets", goJSLoaderScriptName))
	if err != nil {
		t.Fatal(err)
	}
	script := `
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const source = fs.readFileSync(process.argv[1], 'utf8');
let moduleOptions;
function realm(document) {
 const ctx = vm.createContext({console, TextDecoder, TextEncoder, ArrayBuffer, Uint8Array, DataView,
  crypto: {getRandomValues() {}}, performance: {now() {return 0}}, URL, document,
  LLGoModule(options) { moduleOptions = options; options.instantiateWasm({}, () => {}); return new Promise(() => {}); }});
 vm.runInContext(source, ctx);
 return ctx;
}
const ctx = realm();
const decode = vm.runInContext('new TextDecoder()', ctx);
const buffer = new ArrayBuffer(8, {maxByteLength: 16});
const bytes = new Uint8Array(buffer); bytes.set([65,66,67,68]);
assert.equal(decode.decode(buffer), 'ABCD\0\0\0\0');
assert.equal(decode.decode(new Uint8Array(buffer,1,2)), 'BC');
assert.equal(decode.decode(new DataView(buffer,2,2)), 'CD');
const go = vm.runInContext('new Go()', ctx);
assert.ok(go.importObject);
let code;
ctx.fs.open('missing',0,0,error => {code=error.code});
assert.equal(code,'ENOSYS');
const writes = [];
const origWrite = ctx.fs.writeSync.bind(ctx.fs);
ctx.fs.writeSync = (fd, buf) => { writes.push([fd, decode.decode(buf)]); return origWrite(fd, buf); };
moduleOptions.print('hello', 'world');
moduleOptions.printErr('err');
assert.deepEqual(writes, [[1, 'hello world\n'], [2, 'err\n']]);

const late = vm.createContext({console,TextDecoder,TextEncoder,crypto:{getRandomValues(){}},performance:{now(){return 0}},URL});
vm.runInContext(source, late);
assert.throws(() => vm.runInContext('new Go()', late), /missing the bundled LLGoModule factory/);

function taggedFactory(id) {
  return function(options){ options.instantiateWasm({id}, ()=>{}); return {id}; };
}
function bundled(wasmName, id) {
  const name = JSON.stringify(wasmName);
  return 'var LLGoModule = (' + taggedFactory.toString() + ')(' + JSON.stringify(id) + ');\n' +
    'if (typeof exports === "object" && typeof module === "object") module.exports = LLGoModule;\n' +
    '(function(factory) {\n' +
    'globalThis.__llgoFactories = globalThis.__llgoFactories || {};\n' +
    'if (typeof factory !== "function") return;\n' +
    'globalThis.LLGoModule = factory;\n' +
    'globalThis.__llgoFactories[' + name + '] = factory;\n' +
    'try {\n' +
    '  var base = (typeof document !== "undefined" && document.currentScript && document.currentScript.src) ||\n' +
    '    (typeof __filename === "string" ? "file://" + String(__filename).replace(/\\\\/g, "/") : "") ||\n' +
    '    (typeof location !== "undefined" && location.href) || "";\n' +
    '  if (base) globalThis.__llgoFactories[new URL(' + name + ', base).href] = factory;\n' +
    '} catch (e) {}\n' +
    '})(typeof LLGoModule === "function" ? LLGoModule : globalThis.LLGoModule);\n' + source;
}
const multi = vm.createContext({console, TextDecoder, TextEncoder, crypto:{getRandomValues(){}}, performance:{now(){return 0}}, URL});
vm.runInContext(bundled('app.wasm', 'app'), multi);
const firstGo = multi.Go;
vm.runInContext(bundled('foo.wasm', 'foo'), multi);
assert.equal(multi.Go, firstGo);
assert.equal(vm.runInContext('new Go()', multi).importObject.id, 'foo');
assert.equal(vm.runInContext('new Go({ wasm: "app.wasm" })', multi).importObject.id, 'app');
assert.equal(vm.runInContext('new Go({ wasm: "foo.wasm" })', multi).importObject.id, 'foo');
assert.throws(() => vm.runInContext('new Go({ wasm: "missing.wasm" })', multi), /no factory registered for missing.wasm/);

const dirDoc = { currentScript: { src: 'https://example.com/a/wasm_exec.js' } };
const dirs = vm.createContext({console, TextDecoder, TextEncoder, crypto:{getRandomValues(){}}, performance:{now(){return 0}}, URL,
  document: dirDoc, location: { href: 'https://example.com/' }});
vm.runInContext(bundled('app.wasm', 'a'), dirs);
dirDoc.currentScript.src = 'https://example.com/b/wasm_exec.js';
vm.runInContext(bundled('app.wasm', 'b'), dirs);
assert.equal(vm.runInContext('new Go()', dirs).importObject.id, 'b');
assert.throws(() => vm.runInContext('new Go({ wasm: "app.wasm" })', dirs), /no factory registered for app.wasm/);
assert.equal(vm.runInContext('new Go({ wasm: "https://example.com/a/app.wasm" })', dirs).importObject.id, 'a');
assert.equal(vm.runInContext('new Go({ wasm: "a/app.wasm" })', dirs).importObject.id, 'a');
assert.equal(vm.runInContext('new Go({ wasm: "b/app.wasm" })', dirs).importObject.id, 'b');

const pageDoc = { currentScript: { src: 'https://example.com/a/wasm_exec.js' } };
const page = vm.createContext({console, TextDecoder, TextEncoder, crypto:{getRandomValues(){}}, performance:{now(){return 0}}, URL,
  document: pageDoc, location: { href: 'https://example.com/a/index.html' }});
vm.runInContext(bundled('app.wasm', 'a'), page);
pageDoc.currentScript.src = 'https://example.com/b/wasm_exec.js';
vm.runInContext(bundled('app.wasm', 'b'), page);
assert.equal(vm.runInContext('new Go()', page).importObject.id, 'b');
assert.equal(vm.runInContext('new Go({ wasm: "app.wasm" })', page).importObject.id, 'a');
assert.equal(vm.runInContext('new Go({ wasm: "./app.wasm" })', page).importObject.id, 'a');

const os = require('node:os');
const path = require('node:path');
const cjsDir = fs.mkdtempSync(path.join(os.tmpdir(), 'llgo-cjs-'));
fs.writeFileSync(path.join(cjsDir, 'wasm_exec.js'), bundled('cjs.wasm', 'cjs'));
const cjsModule = require(path.join(cjsDir, 'wasm_exec.js'));
assert.equal(typeof cjsModule, 'function');
assert.equal(new Go().importObject.id, 'cjs');
assert.equal(new Go({ wasm: 'cjs.wasm' }).importObject.id, 'cjs');

const prevCwd = process.cwd();
const tree = fs.mkdtempSync(path.join(os.tmpdir(), 'llgo-tree-'));
fs.mkdirSync(path.join(tree, 'a'));
fs.mkdirSync(path.join(tree, 'b'));
fs.writeFileSync(path.join(tree, 'a', 'wasm_exec.js'), bundled('app.wasm', 'na'));
fs.writeFileSync(path.join(tree, 'b', 'wasm_exec.js'), bundled('app.wasm', 'nb'));
try {
  process.chdir(tree);
  require(path.join(tree, 'a', 'wasm_exec.js'));
  require(path.join(tree, 'b', 'wasm_exec.js'));
  assert.equal(new Go({ wasm: 'a/app.wasm' }).importObject.id, 'na');
  assert.equal(new Go({ wasm: './a/app.wasm' }).importObject.id, 'na');
  assert.equal(new Go({ wasm: 'b/app.wasm' }).importObject.id, 'nb');
  assert.equal(new Go({ wasm: './b/app.wasm' }).importObject.id, 'nb');
} finally {
  process.chdir(prevCwd);
}

function runtimeFactory(kind) {
  const module = {
    Asyncify: { currData: null },
    callMain(args) { module.seenCallMain = args; },
    _main(argc) { module.seenArgc = argc; if (kind === 'exit') { const e = new Error('exit'); e.name = 'ExitStatus'; e.status = 7; throw e; } if (kind === 'sleep') { module.Asyncify.currData = 1; setTimeout(() => { module.Asyncify.currData = null; }, 20); } return 0; },
    stackAlloc() { return 16; },
    setValue() {},
    allocateUTF8() { return 32; },
  };
  return function(options) {
    options.instantiateWasm({}, () => {});
    return module;
  };
}
async function runKind(kind, argv) {
  const rt = vm.createContext({console, TextDecoder, TextEncoder, ArrayBuffer, Uint8Array, DataView, WebAssembly,
    crypto: {getRandomValues() {}}, performance: {now() {return 0}}, URL, setTimeout, LLGoModule: runtimeFactory(kind)});
  vm.runInContext(source, rt);
  const go = vm.runInContext('new Go()', rt);
  const records = [];
  go.exit = code => records.push(code);
  if (argv) go.argv = argv;
  const wasm = new Uint8Array([0,97,115,109,1,0,0,0]);
  const {instance} = await WebAssembly.instantiate(wasm);
  const started = Date.now();
  const code = await go.run(instance);
  return {code, records, ms: Date.now()-started, argc: go._module.seenArgc};
}
(async () => {
  const sync = await runKind('sync');
  assert.equal(sync.code, 0);
  assert.deepEqual(sync.records, [0]);
  const exited = await runKind('exit');
  assert.equal(exited.code, 7);
  assert.deepEqual(exited.records, [7]);
  const args = await runKind('sync', ['custom-name', 'hello']);
  assert.equal(args.argc, 2);
  const sleep = await runKind('sleep');
  assert.equal(sleep.code, 0);
  assert.ok(sleep.ms >= 15);
})().catch(err => { console.error(err); process.exit(1); });
`
	cmd := exec.Command(node, "-e", script, loader)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("loader JavaScript: %v\n%s", err, out)
	}
}

func TestGoJSWasmMarker(t *testing.T) {
	if got := goJSWasmMarker([]byte(goJSWasmMarkerPrefix + "foo")); got != "foo" {
		t.Fatalf("marker without newline = %q", got)
	}
	if got := goJSWasmMarker([]byte("not a marker")); got != "" {
		t.Fatalf("unmarked = %q", got)
	}
}
