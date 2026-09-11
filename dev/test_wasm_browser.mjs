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
const write = value => {
	output.push(String(value));
	result.textContent = output.join("\\n");
	if (result.textContent.includes(expected)) result.dataset.status = "success";
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
	result.dataset.status = "failure";
}
setTimeout(() => {
	if (result.dataset.status === "running") result.dataset.status = "failure";
}, 25000);
</script>`;
}

const server = http.createServer(async (request, response) => {
	try {
		const url = new URL(request.url, "http://localhost");
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

await new Promise((resolve, reject) => {
	server.once("error", reject);
	server.listen(0, "127.0.0.1", resolve);
});

try {
	const chrome = await findChrome();
	const { port } = server.address();
	const child = spawn(chrome, [
		"--headless=new",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--no-sandbox",
		"--virtual-time-budget=30000",
		"--dump-dom",
		`http://127.0.0.1:${port}/`,
	]);
	let stdout = "";
	let stderr = "";
	child.stdout.setEncoding("utf8").on("data", chunk => stdout += chunk);
	child.stderr.setEncoding("utf8").on("data", chunk => stderr += chunk);
	const timer = setTimeout(() => child.kill("SIGKILL"), 60000);
	const status = await new Promise((resolve, reject) => {
		child.once("error", reject);
		child.once("close", resolve);
	});
	clearTimeout(timer);
	process.stdout.write(stdout);
	if (status !== 0) {
		process.stderr.write(stderr);
		throw new Error(`headless browser exited with status ${status}`);
	}
	if (!stdout.includes('data-status="success"')) {
		process.stderr.write(stderr);
		throw new Error("WebAssembly browser module did not complete successfully");
	}
	if (!stdout.includes(expected)) {
		throw new Error(`WebAssembly browser output does not contain ${JSON.stringify(expected)}`);
	}
} finally {
	await new Promise(resolve => server.close(resolve));
}
