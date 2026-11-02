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
  let paused, pausedSession, pagePause, stepping, stepped;
  function workerPaused(event, sessionId) {
    workerPauses.add(sessionId);
    if (stepping && sessionId === pausedSession) stepped = event;
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
    const state = await until(async () => {
      const state = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
      if (state.error) throw new Error(state.error);
      return state.paused && state.targets >= workerSessions.length + 1 ? state : null;
    }, 'product coordinator paused every worker');
    assert.ok(state.stopId > 0);
    const again = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
    assert.equal(again.stopId, state.stopId, 'stop ID is stable while stopped');
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
  const optimizedFixture = process.env.LLGO_BROWSER_DEBUG_OPTIMIZED === '1';
  await extension(`(() => {
    const plugin = globalThis.__llgoLanguageExtensionPlugin;
    globalThis.__llgoEvaluations = [];
    const evaluate = plugin.evaluate.bind(plugin);
    plugin.evaluate = async (...args) => {
      try {
        const result = await evaluate(...args);
        if (args[0] === ${JSON.stringify(optimizedFixture ? 'input' : 'cpp_local')} || args[0] === 'value') {
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
  assert.ok(index.variables.every(variable => variable.scope === 'GLOBAL' || variable.ranges?.length),
    'locals without executable scopes must not leak into unrelated frames');
  const goroutineFixture = index.variables.some(v => v.name === 'wasmDebuggerRegistry');
  if (goroutineFixture) {
    browser = await connect({webSocketDebuggerUrl: `ws://127.0.0.1:${port}${browserPath}`});
    browser.on('Debugger.paused', workerPaused);
  }
  const fixture = index.sources.find(s => s.path.endsWith('/fixture.c'));
  const source = fixture || index.sources.find(s => s.path.endsWith('/_wrap/probe.cpp'));
  assert.ok(source, `debug fixture source is indexed: ${JSON.stringify({
    diagnostics: index.diagnostics, sources: index.sources.map(s => s.path),
  })}`);
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
  page.on('Debugger.paused', event => {
    pagePause = event;
    if (stepping && !pausedSession) stepped = event;
    if (event.hitBreakpoints.some(id => breakpoints.some(b => b.id === id))) {
      paused = event;
      pausedSession = undefined;
    }
  });
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
  // The optimized C++ fixture can eliminate cpp_local; input remains live.
  const expectedName = fixture ? 'value' : optimizedFixture ? 'input' : 'cpp_local';
  const expectedValue = fixture ? 42 : optimizedFixture ? 14 : 21;
  let evaluations;
  try {
    await until(async () => {
      evaluations = JSON.parse(await extension('JSON.stringify(globalThis.__llgoEvaluations, (_, value) => typeof value === "bigint" ? String(value) : value)'));
      return evaluations.some(e => e.name === expectedName && e.result?.value === expectedValue);
    }, `real paused ${expectedName}=${expectedValue}`);
  } catch (error) {
    const control = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
    const plugin = await extension(`(() => {
      const p = globalThis.__llgoLanguageExtensionPlugin;
      return {modules: [...p.modules.keys()], calls: p.addRawModuleCalls, error: p.lastAddRawModuleError};
    })()`);
    throw new Error(`${error.message}; evaluations=${JSON.stringify(evaluations)}; control=${JSON.stringify(control)}; plugin=${JSON.stringify(plugin)}; pause=${JSON.stringify(paused.callFrames.slice(0, 2))}`);
  }
  if (goroutineFixture) {
    await pauseWorkers();
    async function runtimeViews() {
      return extension(`(async () => {
        const plugin = globalThis.__llgoLanguageExtensionPlugin;
        const {context, stopId} = globalThis.__llgoLastEvaluation;
        const owned = new Set();
        const root = await plugin.evaluate('debugViews', context, stopId);
        if (!root?.objectId) throw new Error('debugViews is unavailable');
        async function expand(value, depth = 0) {
          if (!value) throw new Error('runtime view returned no value');
          if (value.objectId) owned.add(value.objectId);
          const result = {type: value.type, value: value.value, description: value.description};
          const object = plugin.objects.get(value.objectId);
          if (object?.kind === 'map' && object.length === 49n) {
            result.growing = (await plugin.readNamedUnsigned(plugin.module(context.rawModuleId),
              object.hashType, object.address, object.spec.old_buckets, stopId)) !== 0n;
          }
          if (value.hasChildren && depth < 8) {
            let properties = await plugin.getProperties(value.objectId);
            // Expand the compiler-described captures, never raw code pointers.
            if (value.description?.startsWith('func[')) properties = properties.filter(p => p.name === 'captures');
            result.properties = Object.fromEntries(await Promise.all(
              properties.map(async p => [p.name, await expand(p.value, depth + 1)])));
          }
          return result;
        }
        const tree = await expand(root);
        await plugin.releaseObject(root.objectId);
        const leaked = [...owned].filter(id => plugin.objects.has(id));
        if (leaked.length) throw new Error('runtime view retains released objects: ' + leaked);
        return tree.properties;
      })()`);
    }
    function checkRuntimeViews(views, afterResume = false) {
      const values = value => Object.values(value.properties).map(child => child.value);
      const node = (value, name, numbers) => {
        assert.equal(value.properties.Name.value, name);
        assert.deepEqual(values(value.properties.Values), numbers);
      };
      assert.equal(views.Text.value, 'runtime 世界', JSON.stringify(views.Text));
      for (const record of [views.Imported.properties.value, views.ImportedPointer.properties.value.properties['*']]) {
        assert.equal(record.properties.Name.value, 'imported');
        assert.equal(record.properties.Value.value, 87);
      }
      const first = afterResume ? 55 : 3;
      node(views.Nodes.properties[0], 'runtime 世界', [first, 5, 8]);
      node(views.Nodes.properties[1], 'nested', [13, 21]);
      assert.ok(views.Box.properties?.value, JSON.stringify(views.Box));
      node(views.Box.properties.value, 'runtime 世界', [first, 5, 8]);
      node(views.Pointer.properties.value.properties['*'], 'runtime 世界', [first, 5, 8]);
      node(views.Label.properties.value.properties['*'], 'runtime 世界', [first, 5, 8]);
      assert.equal(views.TypedNil.properties.value.description, 'nil');
      for (const name of ['Nil', 'NilFunction', 'NilMap', 'NilChannel']) {
        assert.equal(views[name].type, 'null', name);
      }
      assert.match(views.Function.description, /^func\[\d+\] main.debugAnswer$/);
      assert.match(views.Closure.description, /^func\[\d+\] main.prepareDebugViews\$1 data=0x[1-9a-f][0-9a-f]*$/);
      node(views.Closure.properties.captures.properties.node.properties['*'].properties['*'],
        'runtime 世界', [first, 5, 8]);
      const captured = value => value.properties.captures.properties;
      node(captured(views.Bound).recv.properties['*'], 'runtime 世界', [first, 5, 8]);
      assert.equal(captured(views.Nested).value.properties['*'].value, 29);
      assert.equal(captured(views.Counter).value.properties['*'].value, afterResume ? 12 : 11);
      assert.equal(captured(views.OtherCounter).value.properties['*'].value, afterResume ? 22 : 21);
      const count = afterResume ? 48 : 49;
      assert.equal(views.Mapping.description, `len=${count}`);
      if (!afterResume) assert.equal(views.Mapping.growing, true, 'inspect a partially evacuated map');
      const entries = views.Mapping.properties;
      assert.equal(Object.keys(entries).length, count * 2);
      const keys = new Set();
      for (let i = 0; i < count; i++) {
        const key = entries[`key[${i}]`].value;
        assert.ok(key >= 0 && key < 49 && (!afterResume || key !== 7));
        keys.add(key);
        node(entries[`value[${i}]`], 'entry', [key, key + 100]);
      }
      assert.equal(keys.size, count, 'map growth must not duplicate entries');
      assert.deepEqual(values(views.Indirect.properties['key[0]']), [7, ...Array(16).fill(0)]);
      assert.deepEqual(values(views.Indirect.properties['value[0]']), [11, 19, ...Array(16).fill(0)]);
      assert.deepEqual([views.Set.properties['key[0]'].value, views.Set.properties['key[1]'].value].sort(), [7, 9]);
      assert.equal(views.Queue.description, `len=${afterResume ? 0 : 3} cap=3 closed`);
      if (!afterResume) {
        node(views.Queue.properties[0], 'runtime 世界', [3, 5, 8]);
        node(views.Queue.properties[1], 'nested', [13, 21]);
        node(views.Queue.properties[2], 'wrapped', [34]);
      }
      assert.equal(views.Signals.description, 'len=2 cap=2 closed');
      assert.equal(Object.keys(views.Signals.properties).length, 2);
      assert.equal(Object.keys(views.Empty.properties).length, 3);
      assert.equal(Object.keys(views.EmptyArray.properties).length, 3);
    }
    checkRuntimeViews(await runtimeViews());
    async function goroutines() {
      return extension(`(async () => {
        const plugin = globalThis.__llgoLanguageExtensionPlugin;
        const {context, stopId} = globalThis.__llgoLastEvaluation;
        const records = await LLGoWasmGoroutines.read(plugin, plugin.module(context.rawModuleId), context, stopId);
        const root = await plugin.evaluate('$goroutines', context, stopId);
        const values = await plugin.getProperties(root.objectId);
        if (values.length !== records.length) throw new Error('goroutine view differs from registry');
        for (const value of values) {
          const record = records.find(record => record.id === value.name);
          const properties = await plugin.getProperties(value.value.objectId);
          const frames = properties.find(p => p.name === 'frames').value;
          for (const child of await plugin.getProperties(frames.objectId)) {
            const frame = record.frames[Number(child.name)];
            if (frame.function !== 'main.debugWaiter') continue;
            const locals = await plugin.getProperties(child.value.objectId);
            const local = locals.find(property => property.name === 'local')?.value;
            if (!local?.objectId) throw new Error('parked array local unavailable: ' + JSON.stringify(locals));
            frame.local = (await plugin.getProperties(local.objectId)).map(property => property.value.value);
          }
        }
        const owned = [...plugin.objects].filter(([, object]) =>
          object.kind === 'snapshot' || object.kind === 'parked-frame').map(([id]) => id);
        await plugin.releaseObject(root.objectId);
        if (owned.some(id => plugin.objects.has(id))) throw new Error('goroutine view retains released objects');
        return records;
      })()`);
    }
    const records = await goroutines();
    const beforeLifecycle = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
    assert.ok(pagePause, 'host page is paused too');
    async function atPausedPage(expression) {
      const result = await page.send('Debugger.evaluateOnCallFrame', {
        callFrameId: pagePause.callFrames[0].callFrameId, expression, returnByValue: true,
      });
      assert.ok(!result.exceptionDetails, JSON.stringify(result));
    }
    await atPausedPage(`globalThis.__llgoLifecycleWorker = new Worker(URL.createObjectURL(
      new Blob(['setInterval(() => {}, 100)'], {type: 'text/javascript'}))); void 0`);
    const withNewWorker = await until(async () => {
      const state = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
      if (state.error) throw new Error(state.error);
      return state.paused && state.targets === beforeLifecycle.targets + 1 ? state : null;
    }, 'new worker joins the coordinated stop');
    assert.ok(withNewWorker.stopId > beforeLifecycle.stopId);
    await atPausedPage('globalThis.__llgoLifecycleWorker.terminate(); delete globalThis.__llgoLifecycleWorker');
    const afterLifecycle = await until(async () => {
      const state = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
      if (state.error) throw new Error(state.error);
      return state.paused && state.targets === beforeLifecycle.targets ? state : null;
    }, 'exited worker leaves the coordinated stop');
    assert.ok(afterLifecycle.stopId > withNewWorker.stopId);
    await atPausedPage(`(() => {
      globalThis.__llgoStartupState = new Int32Array(new SharedArrayBuffer(4));
      globalThis.__llgoStartupWorker = new Worker(URL.createObjectURL(new Blob([
        'onmessage = ({data}) => Atomics.store(new Int32Array(data), 0, 1)'
      ], {type: 'text/javascript'})));
      __llgoStartupWorker.postMessage(__llgoStartupState.buffer);
    })()`);
    await until(async () => {
      const state = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
      if (state.error) throw new Error(state.error);
      return state.paused && state.targets === beforeLifecycle.targets + 1;
    }, 'worker waits at startup until the group resumes');
    await atPausedPage(`(() => {
      if (Atomics.load(__llgoStartupState, 0) !== 0) throw new Error('worker ran during the stop');
    })()`);
    const parked = records.filter(g => g.frames.some(f => f.function === 'main.debugWaiter'));
    const recovered = records.find(g => g.frames.some(f => f.function === 'main.debugRecoveredWaiter'));
    assert.equal(recovered?.state, 'waiting', 'recovered goroutine parks in a later defer');
    assert.ok(recovered.frames.filter(f => f.function === 'main.debugRecoveredWaiter' || f.function === 'main.debugPanicLeaf')
      .every(f => !f.base), 'cleared panic snapshots must not expose retained physical frame addresses');
    assert.equal(parked.length, 2, JSON.stringify(records));
    assert.ok(parked.every(g => g.state === 'waiting'), JSON.stringify(parked));
    assert.deepEqual(parked.map(g => g.frames.filter(f => f.function === 'main.debugWaiter').length).sort(), [4, 5]);
    assert.ok(parked.every(g => g.frames.every(f => f.file && Number(f.line) > 0)), JSON.stringify(parked));
    for (const goroutine of parked) {
      const frames = goroutine.frames.filter(frame => frame.function === 'main.debugWaiter');
      for (const [depth, frame] of frames.entries()) {
        assert.deepEqual(frame.local, Array.from({length: 4}, (_, index) =>
          depth + index * 11 + (index === (depth & 3) ? 100 : 0)), 'parked recursive local after GC and memory growth');
      }
    }
    assert.equal(new Set(parked.map(g => g.id)).size, 2);
    assert.ok(parked.every(g => g.processor !== null), JSON.stringify(parked));
    const main = records.find(g => g.frames.some(f => f.function === 'main.main'));
    assert.equal(main?.processor, '0', JSON.stringify(records));
    if (workerSessions.length > 1) {
      assert.equal(new Set(parked.map(g => g.processor)).size, 2, JSON.stringify(parked));
    }
    stepping = true;
    if (pausedSession) await browser.send('Debugger.stepOver', {}, pausedSession);
    else await page.send('Debugger.stepOver');
    await until(() => stepped, 'selected target steps');
    stepping = false;
    paused = stepped;
    await pauseWorkers();
    const afterStep = await (await fetch(`${sessionURL}__llgo/worker-state`)).json();
    assert.ok(afterStep.stopId > afterLifecycle.stopId, 'step creates a new coordinated stop');
    const oldStopId = await extension('String(globalThis.__llgoLastEvaluation.stopId)');
    paused = null;
    await resumeSelected();
    await until(() => paused, 'goroutine cleanup breakpoint');
    // Let DevTools associate its current stop ID with the new pause.
    await until(async () => (await extension('String(globalThis.__llgoLastEvaluation.stopId)')) !== oldStopId,
      'cleanup stop ID');
    await until(async () => {
      return (await extension(`globalThis.__llgoEvaluations.filter(e => e.name === ${JSON.stringify(expectedName)} && e.result?.value === ${expectedValue}).length`)) >= 2;
    }, 'cleanup C++ local');
    await pauseWorkers();
    checkRuntimeViews(await runtimeViews(), true);
    const after = await goroutines();
    assert.ok(parked.every(g => !after.some(live => live.id === g.id)), JSON.stringify(after));
    assert.ok(!after.some(live => live.id === recovered.id), 'recovered goroutine record is removed');
    console.log('parked locals and deep runtime views survive GC and memory growth; resumed values refresh and exited records disappear');
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
  if (goroutineFixture) {
    await until(async () => {
      const {result} = await page.send('Runtime.evaluate', {
        expression: 'Atomics.load(__llgoStartupState, 0)', returnByValue: true,
      });
      return result.value === 1;
    }, 'group resume releases the startup worker');
    await page.send('Runtime.evaluate', {expression: '__llgoStartupWorker.terminate()'});
  }
  console.log(`source breakpoint, ${expectedName}=${expectedValue}, and program completion passed`);
} finally {
  page.close();
  frontend.close();
  browser?.close();
}
