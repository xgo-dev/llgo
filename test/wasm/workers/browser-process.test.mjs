import assert from "node:assert/strict";
import { once } from "node:events";
import { createServer } from "node:net";
import test from "node:test";
import { spawnBrowser } from "../../../internal/build/testdata/wasm-workers/browser-process.mjs";

test("cleanup stops descendants after the browser launcher exits", {
  skip: process.platform === "win32",
  timeout: 5_000,
}, async t => {
  const server = createServer();
  t.after(() => server.close());
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const connection = once(server, "connection");
  const descendant = `
    require("node:net").connect(${server.address().port}, "127.0.0.1");
    setInterval(() => {}, 1000);
  `;
  const launcher = `
    const child = require("node:child_process").spawn(process.execPath,
      ["-e", ${JSON.stringify(descendant)}],
      { stdio: ["ignore", "ignore", 2] });
    child.unref();
  `;
  const { child, stop } = spawnBrowser(process.execPath, ["-e", launcher]);
  t.after(stop);
  const launcherExit = once(child, "exit");
  const [socket] = await connection;
  t.after(() => socket.destroy());
  const [status] = await launcherExit;
  assert.equal(status, 0);
  assert.equal(child.stderr.destroyed, false, "descendant still owns the stderr pipe");

  // This independent socket closes only when the descendant exits. Destroying
  // the launcher's stderr pipe alone cannot satisfy the regression check.
  const descendantExit = once(socket, "close");
  socket.resume();
  stop();
  await descendantExit;
  assert.equal(child.stderr.destroyed, true);
});
