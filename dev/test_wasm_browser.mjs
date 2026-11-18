import { spawn } from "node:child_process";
import { access, readFile } from "node:fs/promises";
import http from "node:http";
import path from "node:path";
import process from "node:process";

if (process.argv.length < 4) {
	throw new Error("usage: node test_wasm_browser.mjs <module.mjs> <expected-output> [program-args...]");
}

const modulePath = path.resolve(process.argv[2]);
const expected = process.argv[3];
const programArgs = process.argv.slice(4);
const requireExit = process.env.LLGO_BROWSER_REQUIRE_EXIT === "1";
const pclnMode = process.env.LLGO_BROWSER_PCLN;
let pclnRequests = 0;
const root = path.dirname(modulePath);
const moduleName = path.basename(modulePath);
const browserProfile = path.join(root, `.chrome-profile-${process.pid}`);
let resolvePageResult;
const pageResult = new Promise(resolve => resolvePageResult = resolve);

async function findChrome() {
	const candidates = [
		process.env.CHROME,
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	].filter(Boolean);
	for (const candidate of candidates) {
		try {
			await access(candidate);
			return candidate;
		} catch {
			// Try the next standard installation path.
		}
	}
	throw new Error("Chrome or Chromium is required for WebAssembly browser acceptance");
}

function page() {
	return `<!doctype html>
<meta charset="utf-8">
<pre id="result" data-status="running"></pre>
<script type="module">
const result = document.querySelector("#result");
const output = [];
const expected = ${JSON.stringify(expected)};
const requireExit = ${JSON.stringify(requireExit)};
let finished = false;
let successfulExit = false;
const finish = (status, detail) => {
	if (finished) return;
	finished = true;
	result.dataset.status = status;
	const report = String(detail).slice(-4096);
	fetch("/__result?" + new URLSearchParams({ status, detail: report }), { keepalive: true }).catch(() => {});
};
const write = value => {
	output.push(String(value));
	result.textContent = output.join("\\n");
	if (result.textContent.includes(expected) && (!requireExit || successfulExit)) finish("success", result.textContent);
};
for (const method of ["log", "info", "warn", "error"]) {
	const original = console[method].bind(console);
	console[method] = (...values) => {
		write(values.join(" "));
		original(...values);
	};
}
const recordExit = status => {
	if (status !== 0) {
		finish("failure", "exit " + status + ": " + result.textContent);
		return;
	}
	successfulExit = true;
	if (result.textContent.includes(expected)) finish("success", result.textContent);
};
try {
	if (${JSON.stringify(moduleName.endsWith(".wasm"))}) {
		// Raw GoJS keeps Emscripten alive by design. Its supported completion
		// contract is Go.run/Go.exit from the generated wasm_exec.js adapter.
		await new Promise((resolve, reject) => {
			const script = document.createElement("script");
			script.src = "/wasm_exec.js";
			script.onload = resolve;
			script.onerror = () => reject(new Error("failed to load wasm_exec.js"));
			document.head.append(script);
		});
		const go = new Go({ wasm: ${JSON.stringify(`/${moduleName}`)} });
		go.argv = [${JSON.stringify(moduleName)}, ...${JSON.stringify(programArgs)}];
		go.exit = recordExit;
		const loaded = await WebAssembly.instantiateStreaming(fetch(${JSON.stringify(`/${moduleName}`)}), go.importObject);
		await go.run(loaded.instance);
	} else {
		const loaded = await import(${JSON.stringify(`/${moduleName}`)});
		await loaded.default({
			arguments: ${JSON.stringify(programArgs)}, print: write, printErr: write,
			onAbort(reason) { finish("failure", String(reason)); },
			onExit: recordExit,
		});
	}
} catch (error) {
	write(error?.stack || error);
	finish("failure", result.textContent);
}
setTimeout(() => {
	finish("failure", result.textContent || "browser page timed out after 25 seconds");
}, 25000);
</script>`;
}

const server = http.createServer(async (request, response) => {
	try {
		response.setHeader("Cross-Origin-Opener-Policy", "same-origin");
		response.setHeader("Cross-Origin-Embedder-Policy", "require-corp");
		const url = new URL(request.url, "http://localhost");
		if (url.pathname === "/__result") {
			const status = url.searchParams.get("status");
			if (status !== "success" && status !== "failure") {
				response.writeHead(400).end("invalid browser result");
				return;
			}
			response.end("ok");
			resolvePageResult({ pageStatus: status, detail: url.searchParams.get("detail") || "" });
			return;
		}
		if (url.pathname === "/") {
			response.setHeader("Content-Type", "text/html; charset=utf-8");
			response.end(page());
			return;
		}
		const requested = path.resolve(root, `.${decodeURIComponent(url.pathname)}`);
		if (requested !== root && !requested.startsWith(`${root}${path.sep}`)) {
			response.writeHead(403).end();
			return;
		}
		if (pclnMode && requested.endsWith(".pclntab")) pclnRequests++;
		let data = await readFile(requested);
		if (pclnMode && requested.endsWith(".pclntab")) {
			if (pclnMode === "missing") { response.writeHead(404).end(); return; }
			if (pclnMode === "timeout" || pclnMode === "delayed") {
				const timer = setTimeout(() => response.end(data), pclnMode === "timeout" ? 8000 : 500);
				response.once("close", () => clearTimeout(timer));
				return;
			}
			if (pclnMode === "corrupt") data[data.length-1] ^= 1;
			if (pclnMode === "stale") data[64] ^= 1;
			if (pclnMode === "oversized") {
				response.setHeader("Content-Length", (64 << 20) + 1);
				response.end(); return;
			}
		}
		response.setHeader(
			"Content-Type",
			requested.endsWith(".wasm") ? "application/wasm" :
			requested.endsWith(".pclntab") ? "application/octet-stream" : "text/javascript; charset=utf-8",
		);
		response.end(data);
	} catch (error) {
		response.writeHead(error?.code === "ENOENT" ? 404 : 500).end(String(error));
	}
});

function killBrowser(child) {
	if (child.pid === undefined) return;
	if (process.platform !== "win32") {
		try {
			process.kill(-child.pid, "SIGKILL");
			return;
		} catch {
			// Fall back to the direct child if it has already left its group.
		}
	}
	child.kill("SIGKILL");
}

await new Promise((resolve, reject) => {
	server.once("error", reject);
	server.listen(0, "127.0.0.1", resolve);
});

try {
	const chrome = await findChrome();
	const { port } = server.address();
	const child = spawn(chrome, [
		"--headless=new",
		`--user-data-dir=${browserProfile}`,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--no-sandbox",
		"--proxy-server=direct://",
		"--proxy-bypass-list=*",
		`http://127.0.0.1:${port}/`,
	], { detached: process.platform !== "win32" });
	let stdout = "";
	let stderr = "";
	child.stdout.setEncoding("utf8").on("data", chunk => stdout += chunk);
	child.stderr.setEncoding("utf8").on("data", chunk => stderr += chunk);
	const closed = new Promise(resolve => {
		child.once("error", error => resolve({ error }));
		child.once("close", status => resolve({ status, closed: true }));
	});
	let timer;
	const outcome = await Promise.race([
		pageResult,
		closed,
		new Promise(resolve => {
			timer = setTimeout(() => resolve({ timedOut: true }), 60000);
		}),
	]);
	clearTimeout(timer);
	killBrowser(child);
	child.stdout.destroy();
	child.stderr.destroy();
	child.unref();
	if (outcome.timedOut) {
		process.stdout.write(stdout);
		process.stderr.write(stderr);
		throw new Error("headless browser timed out after 60 seconds");
	}
	if (outcome.error) throw outcome.error;
	if (outcome.closed) {
		process.stdout.write(stdout);
		process.stderr.write(stderr);
		throw new Error(`headless browser exited with status ${outcome.status}`);
	}
	if (outcome.pageStatus !== "success") {
		process.stderr.write(stderr);
		throw new Error(`WebAssembly browser module failed: ${outcome.detail}`);
	}
	if (pclnMode && pclnRequests !== (pclnMode === "no-symbols" ? 0 : 1)) {
		throw new Error(`PCLN ${pclnMode}: unexpected request count ${pclnRequests}`);
	}
	process.stdout.write(`${outcome.detail}\n`);
} finally {
	server.closeAllConnections?.();
	await new Promise(resolve => server.close(resolve));
}
