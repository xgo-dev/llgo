import { readFile } from "node:fs/promises";
import { pathToFileURL } from "node:url";

import "./emscripten-node-polyfills.mjs";
import { runEmscriptenModule } from "./emscripten-exit-status.mjs";

if (process.argv.length < 3) {
	throw new Error("usage: node emscripten-runner.mjs <module.mjs> [arguments...]");
}

const moduleURL = pathToFileURL(process.argv[2]);
const loaded = await import(moduleURL);
if (typeof loaded.default !== "function") {
	throw new Error(`${process.argv[2]} does not export an Emscripten module factory`);
}
const wasmURL = new URL(moduleURL);
wasmURL.pathname = wasmURL.pathname.replace(/\.[^/.]+$/, ".wasm");
const moduleOptions = {
	arguments: process.argv.slice(3),
	preRun: [module => {
		if (module.ENV != null) {
			Object.assign(module.ENV, process.env);
		}
	}],
};
try {
	// Browser-only GoJS output deliberately has no Node loader. Supplying the
	// adjacent binary lets this runner validate that same output in CI without
	// changing the generated host contract.
	const wasmBinary = await readFile(wasmURL);
	moduleOptions.instantiateWasm = (imports, receiveInstance) => {
		WebAssembly.instantiate(wasmBinary, imports).then(result => receiveInstance(result.instance));
	};
} catch (error) {
	// Unit-test factories and single-file modules do not have a sibling binary.
	if (error?.code !== "ENOENT") {
		throw error;
	}
}
await runEmscriptenModule(loaded.default, moduleOptions);
