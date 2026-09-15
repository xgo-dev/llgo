import { spawn } from "node:child_process";
import { access, readFile } from "node:fs/promises";
import http from "node:http";
import path from "node:path";
import process from "node:process";

if (process.argv.length !== 4) {
	throw new Error("usage: node test_wasm_browser.mjs <module.mjs> <expected-output>");
}

const modulePath = path.resolve(process.argv[2]);
const expected = process.argv[3];
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
let finished = false;
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
	if (result.textContent.includes(expected)) finish("success", result.textContent);
};
for (const method of ["log", "info", "warn", "error"]) {
	const original = console[method].bind(console);
	console[method] = (...values) => {
		write(values.join(" "));
		original(...values);
	};
}
try {
	const loaded = await import(${JSON.stringify(`/${moduleName}`)});
	await loaded.default({ print: write, printErr: write });
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
		const data = await readFile(requested);
		response.setHeader(
			"Content-Type",
			requested.endsWith(".wasm") ? "application/wasm" : "text/javascript; charset=utf-8",
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
	process.stdout.write(`${outcome.detail}\n`);
} finally {
	server.closeAllConnections?.();
	await new Promise(resolve => server.close(resolve));
}
