import { spawnSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

import "./emscripten-node-polyfills.mjs";
import { runEmscriptenModule } from "./emscripten-exit-status.mjs";

if (process.argv.length < 3) {
	throw new Error("usage: node emscripten-memory64-runner.mjs <module.mjs> [arguments...]");
}

// A memory section whose limits use the memory64 flag. Older Node releases
// expose this behind --experimental-wasm-memory64, while newer releases enable
// it by default and reject the obsolete command-line flag. Probe the engine so
// llgo run works with both families without imposing a Node-version-specific
// emulator command on every user.
const memory64Probe = new Uint8Array([
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	0x05, 0x03, 0x01, 0x04, 0x01,
]);

if (!WebAssembly.validate(memory64Probe)) {
	if (process.env.LLGO_MEMORY64_NODE_RETRY === "1") {
		throw new Error("this Node release does not support WebAssembly Memory64");
	}
	const child = spawnSync(
		process.execPath,
		["--experimental-wasm-memory64", fileURLToPath(import.meta.url), ...process.argv.slice(2)],
		{
			stdio: "inherit",
			env: { ...process.env, LLGO_MEMORY64_NODE_RETRY: "1" },
		},
	);
	if (child.error) {
		throw child.error;
	}
	process.exit(child.status ?? 1);
}

const loaded = await import(pathToFileURL(process.argv[2]));
if (typeof loaded.default !== "function") {
	throw new Error(`${process.argv[2]} does not export an Emscripten module factory`);
}
await runEmscriptenModule(loaded.default, {
	arguments: process.argv.slice(3),
	// Recent Emscripten releases snapshot Module.ENV before preRun. Supplying
	// the object at factory construction keeps os.Getenv consistent while the
	// callback below preserves compatibility with releases that initialize it
	// during preRun.
	ENV: { ...process.env },
	preRun: [module => {
		if (module.ENV != null) {
			Object.assign(module.ENV, process.env);
		}
	}],
});
