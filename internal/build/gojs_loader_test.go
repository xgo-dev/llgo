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
	"runtime"
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

func TestGoJSLoaderSidecars(t *testing.T) {
	output := filepath.Join("dist", "app.mjs")
	if got, want := goJSLoaderSidecar(output), filepath.Join("dist", goJSLoaderScriptName); got != want {
		t.Fatalf("loader = %q, want %q", got, want)
	}
	if got, want := goJSUniqueLoaderSidecar(output), filepath.Join("dist", "app_wasm_exec.js"); got != want {
		t.Fatalf("unique loader = %q, want %q", got, want)
	}
	if got, want := filepath.Join(filepath.Dir(output), wasmFSScriptName), filepath.Join("dist", wasmFSScriptName); got != want {
		t.Fatalf("filesystem host = %q, want %q", got, want)
	}
}

func TestCombineGoJSExec(t *testing.T) {
	got := combineGoJSExec("app.wasm", []byte("factory"), []byte("loader"))
	want := goJSWasmMarkerPrefix + "app.wasm\nfactory\nloader\n"
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
	for _, requested := range []string{"app.wasm"} {
		t.Run(requested, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, requested)
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
				for path, want := range map[string]string{
					goJSLoaderSidecar(driver): combined,
					src:                       "loader",
				} {
					got, err := os.ReadFile(path)
					if err != nil || string(got) != want {
						t.Fatalf("%s = %q, %v; want %q", path, got, err, want)
					}
				}
				if _, err := os.Stat(goJSUniqueLoaderSidecar(driver)); !os.IsNotExist(err) {
					t.Fatalf("unique glue created for single program: %v", err)
				}
				info, err := os.Stat(goJSLoaderSidecar(driver))
				if err != nil {
					t.Fatal(err)
				}
				if runtime.GOOS != "windows" {
					if perm := info.Mode().Perm(); perm != 0o644 {
						t.Fatalf("implementation mode = %o, want 644", perm)
					}
				}
				if _, err := os.Stat(driver); !os.IsNotExist(err) {
					t.Fatalf("intermediate glue remains: %v", err)
				}
				if requested == "app.wasm" {
					got, err := os.ReadFile(output)
					if err != nil || string(got) != "module" {
						t.Fatalf("requested output = %q, %v", got, err)
					}
				}
			}
		})
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
	for _, sidecar := range []string{goJSLoaderScriptName, "app_wasm_exec.js"} {
		for _, source := range []string{"driver", "requested", "module", "sidecar"} {
			for _, link := range []string{"hardlink", "symlink"} {
				t.Run(sidecar+"/"+source+"/"+link, func(t *testing.T) {
					dir := t.TempDir()
					driver := filepath.Join(dir, "app.mjs")
					requested := filepath.Join(dir, "requested.wasm")
					module := filepath.Join(dir, "app.wasm")
					other := goJSUniqueLoaderSidecar(driver)
					if sidecar == filepath.Base(other) {
						other = goJSLoaderSidecar(driver)
					}
					for _, path := range []string{driver, requested, module, other} {
						if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
							t.Fatal(err)
						}
					}
					original := map[string]string{"driver": driver, "requested": requested, "module": module, "sidecar": other}[source]
					linkFile := os.Link
					if link == "symlink" {
						linkFile = os.Symlink
					}
					if err := linkFile(original, filepath.Join(dir, sidecar)); err != nil {
						t.Fatal(err)
					}
					conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: requested}
					ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
					err := publishGoJSLoaderFrom(ctx, driver, false, t.TempDir())
					if err == nil || !strings.Contains(err.Error(), "collides") {
						t.Fatalf("collision error = %v", err)
					}
					for _, path := range []string{driver, requested, module, other, filepath.Join(dir, sidecar)} {
						got, err := os.ReadFile(path)
						if err != nil || string(got) != "original" {
							t.Fatalf("collision changed %s: %q, %v", path, got, err)
						}
					}
				})
			}
		}
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

func TestPublishGoJSLoaderNoop(t *testing.T) {
	if err := publishGoJSLoader(nil, "unused.mjs", false); err != nil {
		t.Fatal(err)
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
				want = "publish GoJS sidecars"
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

func TestGoJSLoaderMatchesOfficialBrowserHostAPI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "targets", goJSLoaderScriptName))
	if err != nil {
		t.Fatal(err)
	}
	loader := string(data)
	for _, want := range []string{
		"global.Go = class",
		"this.argv",
		"this.env = {}",
		"this.exit",
		"this.importObject = imports",
		"this._module.thisProgram = this.argv[0]",
		"noExitRuntime: false",
		"Go.run: WebAssembly.Instance expected",
		"O_WRONLY: -1",
		"fsync(fd, callback) { callback(null); }",
		"error.code = \"ENOSYS\"",
		"global.path = { resolve: (...segments) => segments.join(\"/\") }",
	} {
		if !strings.Contains(loader, want) {
			t.Errorf("wasm_exec.js missing official Go host API fragment %q", want)
		}
	}
}

func TestGoJSLoaderRoutesProgramOutputToStdout(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "targets", goJSLoaderScriptName))
	if err != nil {
		t.Fatal(err)
	}
	loader := string(data)
	for _, want := range []string{
		"print: (...args) => writeOutput(1, args)",
		"printErr: (...args) => writeOutput(2, args)",
		"global.fs.writeSync(fd, encoder.encode(args.map(String).join(\" \") + \"\\n\"))",
	} {
		if !strings.Contains(loader, want) {
			t.Errorf("wasm_exec.js missing stdout routing %q", want)
		}
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

const late = vm.createContext({console,TextDecoder,TextEncoder,crypto:{getRandomValues(){}},performance:{now(){return 0}},URL,
 document:{currentScript:{src:'https://example.com/wasm_exec.js',async:true,dataset:{}},readyState:'complete'},
 location:{href:'https://example.com/'}});
vm.runInContext(source, late);
assert.equal(typeof late.Go, 'function');
assert.throws(() => vm.runInContext('new Go()', late), /missing the bundled LLGoModule factory/);

const factorySource = 'globalThis.LLGoModule = function(options){ options.instantiateWasm({}, ()=>{}); return {}; };';
const bundled = vm.createContext({console,TextDecoder,TextEncoder,crypto:{getRandomValues(){}},performance:{now(){return 0}},URL});
vm.runInContext(factorySource + "\n" + source, bundled);
const goBundled = vm.runInContext('new Go()', bundled);
assert.ok(goBundled.importObject);
assert.equal(source.includes('eval('), false);

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

func TestGoJSWasmMarkerAndWriteFileAtomic(t *testing.T) {
	if got := goJSWasmMarker([]byte(goJSWasmMarkerPrefix + "foo")); got != "foo" {
		t.Fatalf("marker without newline = %q", got)
	}
	if got := goJSWasmMarker([]byte("not a marker")); got != "" {
		t.Fatalf("unmarked = %q", got)
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "nested", "out.js")
	if err := writeFileAtomic(dst, []byte("ok")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o644 {
			t.Fatalf("mode = %o, want 644", perm)
		}
	}
	file := filepath.Join(dir, "notdir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(filepath.Join(file, "x.js"), []byte("y")); err == nil {
		t.Fatal("expected error writing through a file")
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(subdir, []byte("y")); err == nil {
		t.Fatal("expected error renaming onto a directory")
	}
	if runtime.GOOS != "windows" {
		locked := filepath.Join(dir, "locked")
		if err := os.Mkdir(locked, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(locked, 0o755) })
		if err := writeFileAtomic(filepath.Join(locked, "x.js"), []byte("y")); err == nil {
			t.Fatal("expected error creating temp file in unwritable directory")
		}
	}
}

func TestPublishGoJSLoaderVerbose(t *testing.T) {
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
	if err := os.WriteFile(driver, []byte("implementation"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, driver, true, root); err != nil {
		t.Fatal(err)
	}
}

func TestPublishGoJSLoaderDestWriteError(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "targets", goJSLoaderScriptName), []byte("loader"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "app.wasm")
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: output}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	driver := emscriptenDriverOutput(conf, output)
	if err := os.WriteFile(driver, []byte("implementation"), 0o644); err != nil {
		t.Fatal(err)
	}
	shared := goJSLoaderSidecar(driver)
	if err := os.WriteFile(shared, withGoJSWasmMarker("other.wasm", []byte("other")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(goJSUniqueLoaderSidecar(driver), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := publishGoJSLoaderFrom(ctx, driver, false, root); err == nil {
		t.Fatal("expected dest write error")
	}
}

func TestPublishGoJSLoaderUniqueSharedCollision(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "targets", goJSLoaderScriptName), []byte("loader"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "app.wasm")
	conf := &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "js", Goarch: "wasm", OutFile: output}
	ctx := &context{buildConf: conf, crossCompile: crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}}
	driver := emscriptenDriverOutput(conf, output)
	if err := os.WriteFile(driver, []byte("implementation"), 0o644); err != nil {
		t.Fatal(err)
	}
	shared := goJSLoaderSidecar(driver)
	if err := os.WriteFile(shared, withGoJSWasmMarker("other.wasm", []byte("other")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(shared, goJSUniqueLoaderSidecar(driver)); err != nil {
		t.Fatal(err)
	}
	err := publishGoJSLoaderFrom(ctx, driver, false, root)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("collision error = %v", err)
	}
}

func TestChooseGoJSLoaderDestReadError(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, goJSLoaderScriptName)
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{OutFile: filepath.Join(dir, "app.wasm")}}
	if _, err := chooseGoJSLoaderDest(ctx, filepath.Join(dir, "app.mjs"), filepath.Join(dir, "app_wasm_exec.js"), shared); err == nil {
		t.Fatal("expected read error for directory shared glue")
	}
}
