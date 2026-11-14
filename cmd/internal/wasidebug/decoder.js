// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

// A one-command bridge to the same decoders used by Chrome. Only the stopped
// LLDB process supplies memory; this process never resumes or evaluates code.
const fs = require('node:fs');
const readline = require('node:readline');
require('./goroutines.js');
require('./plugin.js');

const lines = readline.createInterface({input: process.stdin})[Symbol.asyncIterator]();
const send = value => process.stdout.write(JSON.stringify(value, (_, v) => typeof v === 'bigint' ? String(v) : v) + '\n');
const receive = async () => {
  const line = await lines.next();
  if (line.done) throw new Error('LLDB memory transport closed');
  return JSON.parse(line.value);
};

(async () => {
  const request = await receive();
  const index = JSON.parse(fs.readFileSync(__dirname + '/index.json'));
  const schema = JSON.parse(fs.readFileSync(__dirname + '/schema.json'));
  const bytes = fs.readFileSync(__dirname + '/module.wasm');
  const services = {getWasmLinearMemory: async (address, length, stopId) => {
    send({read: {address, length, stopId}});
    const result = await receive();
    if (result.error) throw new Error(result.error);
    return Uint8Array.from(Buffer.from(result.bytes, 'base64')).buffer;
  }};
  const fetcher = async url => ({ok: true, status: 200,
    json: async () => url.endsWith('debug-index.json') ? index : schema});
  const plugin = new globalThis.LLGoLanguageExtension.LLGoLanguageExtensionPlugin(services, fetcher);
  await plugin.addRawModule('wasi', '', {url: 'http://llgo.invalid/module.wasm', code: bytes});
  const module = plugin.modules.get('wasi');
  if (!module) throw new Error('missing LLGo debugger record');
  const context = {rawModuleId: 'wasi', codeOffset: -1};
  let value;
  if (request.type) {
    value = await plugin.remoteObject(module, module.types.get(request.type),
      request.address !== undefined ? {kind: 'address', value: BigInt(request.address)} :
        {kind: 'value', value: BigInt(request.value)}, request.stopId);
  } else {
    value = await plugin.evaluate(request.expression, context, request.stopId);
  }
  if (!value) throw new Error('variable is unavailable or not in scope');
  let remaining = 1024;
  const expand = async (value, depth) => {
    if (--remaining < 0) throw new Error('runtime view exceeds 1024 values; use a smaller depth');
    const result = {...value};
    delete result.objectId;
    if (depth && value.objectId) {
      result.children = [];
      for (const child of await plugin.getProperties(value.objectId)) {
        result.children.push({name: child.name, value: await expand(child.value, depth - 1)});
      }
    }
    return result;
  };
  try {
    send({result: await expand(value, request.depth)});
  } finally {
    if (value.objectId) await plugin.releaseObject(value.objectId);
  }
})().catch(error => {send({error: String(error.message || error)}); process.exitCode = 1;})
  .finally(() => process.stdin.destroy());
