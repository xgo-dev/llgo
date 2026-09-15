import { readFile } from "node:fs/promises";
import { pathToFileURL } from "node:url";

import "./emscripten-node-polyfills.mjs";
import { runEmscriptenModule } from "./emscripten-exit-status.mjs";

const nodeProcess = globalThis.process;
const browserOnly = nodeProcess.argv[2] === "--browser-only";
const moduleArg = browserOnly ? 3 : 2;
if (nodeProcess.argv.length <= moduleArg) {
	throw new Error("usage: node emscripten-runner.mjs [--browser-only] <module.mjs> [arguments...]");
}

const moduleURL = pathToFileURL(nodeProcess.argv[moduleArg]);
const loaded = await import(moduleURL);
if (typeof loaded.default !== "function") {
	throw new Error(`${nodeProcess.argv[moduleArg]} does not export an Emscripten module factory`);
}
const wasmURL = new URL(moduleURL);
wasmURL.pathname = wasmURL.pathname.replace(/\.[^/.]+$/, ".wasm");
const moduleOptions = {
	arguments: nodeProcess.argv.slice(moduleArg + 1),
	preRun: [module => {
		if (module.ENV != null) {
			Object.assign(module.ENV, nodeProcess.env);
		}
	}],
};
let rejectInstantiation;
const instantiationFailure = new Promise((_, reject) => {
	rejectInstantiation = reject;
});
try {
	// Browser-only GoJS output deliberately has no Node loader. Supplying the
	// adjacent binary lets this runner validate that same output in CI without
	// changing the generated host contract.
	const wasmBinary = await readFile(wasmURL);
	moduleOptions.instantiateWasm = (imports, receiveInstance) => {
		WebAssembly.instantiate(wasmBinary, imports).then(
			result => receiveInstance(result.instance),
			rejectInstantiation,
		);
	};
} catch (error) {
	// Unit-test factories and single-file modules do not have a sibling binary.
	if (error?.code !== "ENOENT") {
		throw error;
	}
}
if (browserOnly) {
	// Raw GOOS=js GOARCH=wasm output intentionally excludes Emscripten's Node
	// host. Model a browser before invoking even unoptimized glue, whose
	// environment assertions run before instantiateWasm can supply the binary.
	globalThis.window ??= globalThis;
	globalThis.process = undefined;
}
await Promise.race([
	runEmscriptenModule(loaded.default, moduleOptions),
	instantiationFailure,
]);
