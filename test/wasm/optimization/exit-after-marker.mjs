// A deferred Go success marker can precede a fatal runtime exit. The browser
// harness must reject that sequence when it is checking a pthread program.
export default async function (options) {
	if (options.arguments.join(",") !== "workers") throw new Error("program arguments were lost");
	options.print("wasm deferred marker");
	options.onExit(2);
}
