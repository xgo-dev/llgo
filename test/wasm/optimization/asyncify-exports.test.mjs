import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import vm from "node:vm";

const shim = fs.readFileSync(new URL("../../../targets/emscripten-asyncify-exports.js", import.meta.url), "utf8");
const names = ["start_unwind", "stop_unwind", "start_rewind", "stop_rewind"];

for (const form of ["absent", "function", "array"]) {
	test(`preserves ${form} preRun hook`, () => {
		let called = 0;
		const callback = () => called++;
		const Module = form === "absent" ? {} : { preRun: form === "function" ? callback : [callback] };
		const wasmExports = Object.fromEntries(names.map(name => [`asyncify_${name}`, () => {}]));
		vm.runInNewContext(shim, { Module, wasmExports });
		for (const hook of Module.preRun) hook();
		assert.equal(called, form === "absent" ? 0 : 1);
		delete wasmExports.asyncify_stop_unwind;
		assert.throws(() => Module.preRun[0](), /missing Asyncify control exports/);
	});
}
