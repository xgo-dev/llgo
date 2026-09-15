import assert from "node:assert/strict";
import { pathToFileURL } from "node:url";
import "../targets/emscripten-node-polyfills.mjs";
import { runEmscriptenModule } from "../targets/emscripten-exit-status.mjs";

const [modulePath, mode, operation] = process.argv.slice(2);
assert.ok(modulePath, "usage: test_wasm_js_boundary.mjs <module.mjs> <return|throw|exit-0|exit-7> <Call|Invoke|New>");
assert.ok(["return", "throw", "exit-0", "exit-7"].includes(mode));
assert.ok(["Call", "Invoke", "New"].includes(operation));
const { default: factory } = await import(pathToFileURL(modulePath));
let called = false;
await runEmscriptenModule(factory, {
  preRun: [module => {
    const expected = { code: "ENOENT", message: "host boundary probe" };
    globalThis.llgoHostMode = mode;
    globalThis.llgoHostOperation = operation;
    globalThis.llgoHostExpected = expected;
    globalThis.llgoHostProbe = function (argument) {
      called = true;
      assert.equal(argument, expected, "call lost an argument before entering JS");
      if (mode.startsWith("exit-")) {
        console.log("wasm host exit reached");
        module._emscripten_force_exit(Number(mode.slice(5)));
        throw new Error("emscripten_force_exit returned");
      }
      const before = module.memory.buffer.byteLength;
      assert.equal(module._emscripten_resize_heap(before + 65536), true);
      assert.ok(module.memory.buffer.byteLength > before, "probe did not grow Wasm memory");
      if (mode === "throw") throw expected;
      return expected;
    };
  }],
});
assert.ok(called, "module did not call the host probe");
assert.equal(process.exitCode ?? 0, mode === "exit-7" ? 7 : 0);
console.log("wasm host boundary runner ok");
