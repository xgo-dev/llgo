import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawn } from "node:child_process";

const [browser, url] = process.argv.slice(2);
if (!browser || !url) {
  throw new Error("usage: node browser-runner.mjs <chrome> <url>");
}

const deadline = Date.now() + 30_000;
const profile = await mkdtemp(join(tmpdir(), "llgo-wasm-chrome-"));
const chrome = spawn(browser, [
  "--headless=new",
  "--no-sandbox",
  "--disable-gpu",
  "--disable-dev-shm-usage",
  "--no-first-run",
  `--user-data-dir=${profile}`,
  "--remote-debugging-port=0",
  "about:blank",
], { stdio: ["ignore", "ignore", "pipe"] });
const chromeExit = new Promise(resolve => chrome.once("exit", resolve));

let chromeLog = "";
chrome.stderr.setEncoding("utf8");
chrome.stderr.on("data", chunk => {
  chromeLog = (chromeLog + chunk).slice(-64 * 1024);
});

const delay = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds));

async function activePort() {
  const path = join(profile, "DevToolsActivePort");
  while (Date.now() < deadline) {
    if (chrome.exitCode !== null) {
      throw new Error(`Chrome exited before DevTools was ready:\n${chromeLog}`);
    }
    try {
      const lines = (await readFile(path, "utf8")).trim().split("\n");
      if (lines[0]) {
        return Number(lines[0]);
      }
    } catch (error) {
      if (error.code !== "ENOENT") {
        throw error;
      }
    }
    await delay(50);
  }
  throw new Error(`Chrome did not expose DevTools:\n${chromeLog}`);
}

async function connect(webSocketURL) {
  const socket = new WebSocket(webSocketURL);
  await new Promise((resolve, reject) => {
    socket.addEventListener("open", resolve, { once: true });
    socket.addEventListener("error", reject, { once: true });
  });

  let nextID = 1;
  const pending = new Map();
  const diagnostics = [];
  socket.addEventListener("message", event => {
    const message = JSON.parse(event.data);
    if (message.id !== undefined) {
      const resolve = pending.get(message.id);
      pending.delete(message.id);
      resolve?.(message);
      return;
    }
    if (message.method === "Runtime.consoleAPICalled") {
      diagnostics.push(message.params.args.map(arg => arg.value ?? arg.description).join(" "));
    } else if (message.method === "Runtime.exceptionThrown") {
      const exception = message.params.exceptionDetails;
      diagnostics.push(exception.exception?.description ?? exception.text);
    }
  });

  return {
    diagnostics,
    socket,
    send(method, params = {}) {
      return new Promise(resolve => {
        const id = nextID++;
        pending.set(id, resolve);
        socket.send(JSON.stringify({ id, method, params }));
      });
    },
  };
}

let client;
try {
  const port = await activePort();
  const response = await fetch(
    `http://127.0.0.1:${port}/json/new?${encodeURIComponent("about:blank")}`,
    { method: "PUT" },
  );
  if (!response.ok) {
    throw new Error(`Chrome refused the test page: HTTP ${response.status}`);
  }
  const target = await response.json();
  client = await connect(target.webSocketDebuggerUrl);
  await client.send("Page.enable");
  await client.send("Runtime.enable");
  // Create the target first, attach DevTools, and then navigate explicitly.
  // Asking /json/new to navigate races target attachment on some Chrome/macOS
  // combinations and can leave the client evaluating the initial blank page.
  const navigation = await client.send("Page.navigate", { url });
  if (navigation.error || navigation.result?.errorText) {
    throw new Error(
      `Chrome failed to navigate: ${navigation.error?.message ?? navigation.result.errorText}`,
    );
  }

  let state = { result: "running", text: "page did not initialize", url: "about:blank" };
  while (Date.now() < deadline) {
    const reply = await client.send("Runtime.evaluate", {
      expression: `JSON.stringify({
        url: location.href,
        result: document.body?.dataset.result,
        text: document.querySelector("#status")?.textContent,
      })`,
      returnByValue: true,
    });
    const value = reply.result?.result?.value;
    if (value) {
      const observed = JSON.parse(value);
      state = { ...state, ...observed };
      if (state.result === "pass") {
        process.stdout.write(`${state.text}\n`);
        break;
      }
      if (state.result === "fail") {
        throw new Error(state.text);
      }
    }
    await delay(100);
  }
  if (state.result !== "pass") {
    throw new Error(
      `browser test timed out at ${state.url} (${state.text})\n${client.diagnostics.join("\n")}`,
    );
  }
} catch (error) {
  process.stderr.write(`${error.stack ?? error}\n${chromeLog}`);
  process.exitCode = 1;
} finally {
  client?.socket.close();
  if (chrome.exitCode === null && chrome.signalCode === null) {
    chrome.kill("SIGTERM");
    await Promise.race([chromeExit, delay(2_000)]);
    if (chrome.exitCode === null && chrome.signalCode === null) {
      chrome.kill("SIGKILL");
      await chromeExit;
    }
  }
  // Chrome's child processes can still flush profile files after its main
  // process exits. Node retries ENOTEMPTY for a recursive removal here.
  await rm(profile, { recursive: true, force: true, maxRetries: 20, retryDelay: 100 });
}
