// Exercise the installed extension against real paused Wasm state. Node 22+
// supplies WebSocket; no separate browser automation dependency is needed.
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

const [profile, sessionURL] = process.argv.slice(2);
const [port, browserPath] = (await readFile(`${profile}/DevToolsActivePort`, 'utf8')).split('\n');
const targets = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
const pageTarget = targets.find(t => t.type === 'page' && t.url.startsWith(sessionURL));
const devtoolsTarget = targets.find(t => t.url.startsWith('chrome-extension://') && t.url.endsWith('/devtools.html'));
assert.ok(pageTarget && devtoolsTarget, JSON.stringify(targets));

async function connect(target) {
  const socket = new WebSocket(target.webSocketDebuggerUrl);
  const pending = new Map(), listeners = new Map();
  let sequence = 0;
  socket.onmessage = ({data}) => {
    const message = JSON.parse(data);
    if (message.id) {
      const request = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) request.reject(new Error(JSON.stringify(message.error)));
      else request.resolve(message.result);
    } else {
      for (const listener of listeners.get(message.method) || []) listener(message.params, message.sessionId);
    }
  };
  await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
  return {
    send(method, params = {}, sessionId) {
      return new Promise((resolve, reject) => {
        const id = ++sequence;
        pending.set(id, {resolve, reject});
        socket.send(JSON.stringify({id, method, params, sessionId}));
      });
    },
    on(method, callback) {
      if (!listeners.has(method)) listeners.set(method, []);
      listeners.get(method).push(callback);
    },
    close() { socket.close(); },
  };
}

async function until(probe, description) {
  for (let attempt = 0; attempt < 200; ++attempt) {
    const value = await probe();
    if (value) return value;
    await new Promise(resolve => setTimeout(resolve, 50));
  }
  throw new Error(`timed out: ${description}`);
}

const page = await connect(pageTarget);
const frontend = await connect(devtoolsTarget);
let browser;
try {
  const workerSessions = [], workerPauses = new Set();
  const breakpoints = [];
  let paused, pausedSession;
  function workerPaused(event, sessionId) {
    workerPauses.add(sessionId);
    if (event.hitBreakpoints.some(id => breakpoints.some(b => b.id === id))) {
      paused = event;
      pausedSession = sessionId;
    }
  }
  async function connectWorkers() {
    if (!workerSessions.length) {
      const {targetInfos} = await browser.send('Target.getTargets');
      for (const worker of targetInfos.filter(t => t.type === 'worker' &&
          (t.url.startsWith(sessionURL) || t.url.startsWith(`blob:${sessionURL}`)))) {
        const {sessionId} = await browser.send('Target.attachToTarget', {targetId: worker.targetId, flatten: true});
        workerSessions.push(sessionId);
        await browser.send('Debugger.enable', {}, sessionId);
      }
    }
  }
  async function pauseWorkers() {
    for (const sessionId of workerSessions) {
      if (!workerPauses.has(sessionId)) await browser.send('Debugger.pause', {}, sessionId);
    }
    await until(() => workerSessions.every(id => workerPauses.has(id)), 'all worker targets paused');
  }
  async function resumeWorkers() {
    for (const sessionId of workerSessions) {
      if (sessionId !== pausedSession) {
        await browser.send('Debugger.resume', {}, sessionId);
        workerPauses.delete(sessionId);
      }
    }
  }
  async function resumeSelected() {
    if (pausedSession) {
      await browser.send('Debugger.resume', {}, pausedSession);
      workerPauses.delete(pausedSession);
    } else await page.send('Debugger.resume');
  }
  const contexts = [];
  frontend.on('Runtime.executionContextCreated', ({context}) => contexts.push(context));
  await frontend.send('Runtime.enable');
  const contextID = await until(async () => {
    for (const context of contexts) {
      const {result} = await frontend.send('Runtime.evaluate', {
        contextId: context.id, expression: 'typeof globalThis.__llgoLanguageExtensionPlugin',
      });
      if (result.value === 'object') return context.id;
    }
  }, 'extension execution context');
  async function extension(expression) {
    const result = await frontend.send('Runtime.evaluate', {
      contextId: contextID, expression, awaitPromise: true, returnByValue: true,
    });
    if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails));
    return result.result.value;
  }
  await extension(`(() => {
    const plugin = globalThis.__llgoLanguageExtensionPlugin;
    globalThis.__llgoEvaluations = [];
    const evaluate = plugin.evaluate.bind(plugin);
    plugin.evaluate = async (...args) => {
      try {
        const result = await evaluate(...args);
        if (args[0] === 'cpp_local' || args[0] === 'value') {
          globalThis.__llgoLastEvaluation = {context: args[1], stopId: args[2]};
        }
        globalThis.__llgoEvaluations.push({name: args[0], result});
        return result;
      } catch (error) {
        globalThis.__llgoEvaluations.push({name: args[0], error: String(error)});
        throw error;
      }
    };
  })()`);

  const index = await (await fetch(`${sessionURL}__llgo/debug-index.json`)).json();
  const goroutineFixture = index.variables.some(v => v.name === 'wasmDebuggerRegistry');
  if (goroutineFixture) {
    browser = await connect({webSocketDebuggerUrl: `ws://127.0.0.1:${port}${browserPath}`});
    browser.on('Debugger.paused', workerPaused);
  }
  const fixture = index.sources.find(s => s.path.endsWith('/fixture.c'));
  const source = fixture || index.sources.find(s => s.path.endsWith('/_wrap/probe.cpp'));
  assert.ok(source, 'debug fixture source is indexed');
  const lineNumber = fixture ? 2 : 4; // Return after local initialization, zero-based.
  const mapping = await extension(`(async () => {
    const plugin = globalThis.__llgoLanguageExtensionPlugin;
    const [rawModuleId] = plugin.modules.keys();
    return plugin.sourceLocationToRawLocation({rawModuleId,
      sourceFileURL: ${JSON.stringify(new URL(source.url, sessionURL).href)},
      lineNumber: ${lineNumber}, columnNumber: 0});
  })()`);
  assert.ok(mapping.length, 'extension maps the source breakpoint');
  const scripts = [];
  page.on('Debugger.scriptParsed', script => scripts.push(script));
  page.on('Debugger.paused', event => { paused = event; pausedSession = undefined; });
  await page.send('Debugger.enable');
  const wasm = await until(() => scripts.find(s => s.scriptLanguage === 'WebAssembly'), 'Wasm script');
  const breakpoint = await page.send('Debugger.setBreakpoint', {location: {
    scriptId: wasm.scriptId, lineNumber: 0,
    columnNumber: mapping[0].startOffset + wasm.codeOffset,
  }});
  breakpoints.push({id: breakpoint.breakpointId});
  if (goroutineFixture) {
    await connectWorkers();
    for (const sessionId of workerSessions) {
      const result = await browser.send('Debugger.setBreakpointByUrl', {
        urlRegex: '^wasm://|\\.wasm$', lineNumber: 0,
        columnNumber: mapping[0].startOffset + wasm.codeOffset,
      }, sessionId);
      breakpoints.push({id: result.breakpointId, sessionId});
    }
  }
  // Do not await execution: it will wait while the debugger is paused.
  const run = page.send('Runtime.evaluate', {expression: 'globalThis.__llgoDebugRun()', awaitPromise: true});
  await until(() => paused, 'source breakpoint');
  assert.ok(paused.hitBreakpoints.some(id => breakpoints.some(b => b.id === id)), JSON.stringify(paused));
  const expectedName = fixture ? 'value' : 'cpp_local';
  const expectedValue = fixture ? 42 : 21;
  let evaluations;
  try {
    await until(async () => {
      evaluations = JSON.parse(await extension('JSON.stringify(globalThis.__llgoEvaluations, (_, value) => typeof value === "bigint" ? String(value) : value)'));
      return evaluations.some(e => e.name === expectedName && e.result?.value === expectedValue);
    }, `real paused ${expectedName}=${expectedValue}`);
  } catch (error) {
    throw new Error(`${error.message}; evaluations=${JSON.stringify(evaluations)}; pause=${JSON.stringify(paused.callFrames.slice(0, 2))}`);
  }
  if (goroutineFixture) {
    await pauseWorkers();
    async function goroutines() {
      return extension(`(async () => {
        const plugin = globalThis.__llgoLanguageExtensionPlugin;
        const {context, stopId} = globalThis.__llgoLastEvaluation;
        const records = await LLGoWasmGoroutines.read(plugin, plugin.module(context.rawModuleId), context, stopId);
        const root = await plugin.evaluate('$goroutines', context, stopId);
        const values = await plugin.getProperties(root.objectId);
        if (values.length !== records.length) throw new Error('goroutine view differs from registry');
        for (const value of values) {
          const properties = await plugin.getProperties(value.value.objectId);
          const frames = properties.find(p => p.name === 'frames').value;
          await plugin.getProperties(frames.objectId);
        }
        const owned = [...plugin.objects].filter(([, object]) => object.kind === 'snapshot').map(([id]) => id);
        await plugin.releaseObject(root.objectId);
        if (owned.some(id => plugin.objects.has(id))) throw new Error('goroutine view retains released objects');
        return records;
      })()`);
    }
    const records = await goroutines();
    const parked = records.filter(g => g.frames.some(f => f.function === 'main.debugWaiter'));
    assert.equal(parked.length, 2, JSON.stringify(records));
    assert.ok(parked.every(g => g.state === 'waiting'), JSON.stringify(parked));
    assert.deepEqual(parked.map(g => g.frames.filter(f => f.function === 'main.debugWaiter').length).sort(), [4, 5]);
    assert.ok(parked.every(g => g.frames.every(f => f.file && Number(f.line) > 0)), JSON.stringify(parked));
    assert.equal(new Set(parked.map(g => g.id)).size, 2);
    assert.ok(parked.every(g => g.processor !== null), JSON.stringify(parked));
    const main = records.find(g => g.frames.some(f => f.function === 'main.main'));
    assert.equal(main?.processor, '0', JSON.stringify(records));
    if (workerSessions.length > 1) {
      assert.equal(new Set(parked.map(g => g.processor)).size, 2, JSON.stringify(parked));
    }
    const oldStopId = await extension('String(globalThis.__llgoLastEvaluation.stopId)');
    paused = null;
    await resumeWorkers();
    await resumeSelected();
    await until(() => paused, 'goroutine cleanup breakpoint');
    // Let DevTools associate its current stop ID with the new pause.
    await until(async () => (await extension('String(globalThis.__llgoLastEvaluation.stopId)')) !== oldStopId,
      'cleanup stop ID');
    await until(async () => {
      return (await extension(`globalThis.__llgoEvaluations.filter(e => e.name === ${JSON.stringify(expectedName)} && e.result?.value === ${expectedValue}).length`)) >= 2;
    }, 'cleanup C++ local');
    await pauseWorkers();
    const after = await goroutines();
    assert.ok(parked.every(g => !after.some(live => live.id === g.id)), JSON.stringify(after));
    console.log('parked logical stacks survive GC; exited goroutine records are removed');
    await resumeWorkers();
  }
  for (const breakpoint of breakpoints) {
    if (breakpoint.sessionId) await browser.send('Debugger.removeBreakpoint', {breakpointId: breakpoint.id}, breakpoint.sessionId);
    else await page.send('Debugger.removeBreakpoint', {breakpointId: breakpoint.id});
  }
  await resumeSelected();
  await run;
  if (fixture) {
    const {result} = await page.send('Runtime.evaluate', {
      expression: 'globalThis.__llgoDebugStatus', returnByValue: true,
    });
    assert.deepEqual(result.value, {state: 'exited', code: 0});
  } else {
    // An Emscripten fiber can keep the browser runtime alive after Go main
    // returns. Require the real Go completion sentinel without inventing exit.
    let completion;
    try {
      await until(async () => {
        const {result} = await page.send('Runtime.evaluate', {
          expression: '({status: globalThis.__llgoDebugStatus, output: document.getElementById("output").textContent})',
          returnByValue: true,
        });
        completion = result.value;
        assert.notEqual(result.value.status.state, 'error', JSON.stringify(result.value));
        return result.value.output.includes('wasm debug ok');
      }, 'Go completion output');
    } catch (error) {
      throw new Error(`${error.message}; completion=${JSON.stringify(completion)}; pause=${JSON.stringify(paused?.callFrames.slice(0, 2))}`);
    }
  }
  console.log(`source breakpoint, ${expectedName}=${expectedValue}, and program completion passed`);
} finally {
  page.close();
  frontend.close();
  browser?.close();
}
