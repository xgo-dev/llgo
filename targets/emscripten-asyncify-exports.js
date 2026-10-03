// Asyncify adds these controls after wasm-ld, so EXPORTED_FUNCTIONS cannot
// name them. MetaDCE can miss their generated references (observed with
// Emscripten 6.0.8's EXPORT_ALL chained assignments). Explicit reads in this
// observable preRun check create usage edges that keep the controls alive.
// Apply the check to every Asyncify configuration, including workers with
// EXPORT_ALL=0, without changing the public Module exports or user hooks.
function llgoCheckAsyncifyExports() {
	var startUnwind = wasmExports["asyncify_start_unwind"];
	var stopUnwind = wasmExports["asyncify_stop_unwind"];
	var startRewind = wasmExports["asyncify_start_rewind"];
	var stopRewind = wasmExports["asyncify_stop_rewind"];
	if (typeof startUnwind !== "function" || typeof stopUnwind !== "function" ||
		typeof startRewind !== "function" || typeof stopRewind !== "function") {
		throw new Error("LLGo: missing Asyncify control exports");
	}
}
var llgoPreRun = Module["preRun"] || [];
if (typeof llgoPreRun === "function") llgoPreRun = [llgoPreRun];
Module["preRun"] = [llgoCheckAsyncifyExports, ...llgoPreRun];
