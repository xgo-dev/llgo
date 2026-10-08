// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

(() => {
  'use strict';

  const LIMIT = 4096;

  function resolve(module, type) {
    const seen = new Set();
    while (type?.kind === 'typedef') {
      if (seen.has(type.id)) throw new Error('cyclic debugger type');
      seen.add(type.id);
      type = module.types.get(type.elem);
    }
    return type;
  }

  function fieldType(module, type, name) {
    const field = resolve(module, type)?.fields?.find(field => field.name === name);
    if (!field) throw new Error(`missing debugger field ${name}`);
    return resolve(module, module.types.get(field.type));
  }

  function pointee(module, type, name) {
    const pointer = fieldType(module, type, name);
    if (pointer?.kind !== 'pointer') throw new Error(`debugger field ${name} is not a pointer`);
    return resolve(module, module.types.get(pointer.elem));
  }

  async function read(plugin, module, context, stopId) {
    const spec = module.layout?.wasm_goroutine;
    if (!spec) return null;
    const variable = module.index.variables.find(variable => variable.scope === 'GLOBAL' &&
        variable.name === spec.registry_symbol);
    if (!variable) return null;
    const located = await plugin.variableLocation(variable, context.codeOffset, module, stopId);
    if (located?.kind !== 'address') throw new Error('Wasm goroutine registry has no address');
    const registry = resolve(module, module.types.get(variable.type));
    // Fetch fixed structs and the frame array once; scalar fields and string
    // headers are decoded locally instead of making a transport call per field.
    const fields = (type, names) => Object.fromEntries(names.map(name => {
      const field = type?.fields?.find(field => field.name === name);
      if (!field) throw new Error(`missing debugger field ${name}`);
      const valueType = resolve(module, module.types.get(field.type));
      return [name, {...field, size: valueType?.kind === 'pointer' ? module.record.pointer_size : valueType?.size}];
    }));
    const unsigned = (bytes, field, base = 0) => {
      const offset = base + field.offset;
      if (!Number.isInteger(field.size) || field.size < 1 || field.size > 8 ||
          !Number.isInteger(offset) || offset < 0 || offset + field.size > bytes.length) {
        throw new Error(`unreadable debugger field ${field.name}`);
      }
      let value = 0n;
      for (let index = 0; index < field.size; ++index) value |= BigInt(bytes[offset + index]) << BigInt(index * 8);
      return value;
    };
    const memory = async (address, size) => {
      if (address < 0n || address > BigInt(Number.MAX_SAFE_INTEGER) ||
          !Number.isInteger(size) || size < 0 || size > LIMIT * 256) throw new Error('invalid debugger memory range');
      const bytes = new Uint8Array(await plugin.languageServices.getWasmLinearMemory(Number(address), size, stopId));
      if (bytes.length !== size) throw new Error('incomplete debugger memory read');
      return bytes;
    };
    const registryFields = fields(registry, [spec.version, spec.epoch, spec.head]);
    const registryBytes = await memory(located.value, registry.size);
    const version = unsigned(registryBytes, registryFields[spec.version]);
    if (version !== BigInt(spec.registry_version)) throw new Error(`unsupported Wasm goroutine registry ${version}`);
    const epoch = unsigned(registryBytes, registryFields[spec.epoch]);
    if (epoch & 1n) throw new Error('goroutine registry is changing; pause all workers outside runtime updates');
    let nodeAddress = unsigned(registryBytes, registryFields[spec.head]);
    const node = pointee(module, registry, spec.head);
    const g = pointee(module, node, spec.goroutine);
    const caller = pointee(module, node, spec.callers);
    const stackField = caller.fields.find(field => field.name === spec.stack);
    const stack = fieldType(module, caller, spec.stack);
    const frame = pointee(module, stack, module.layout.slice.data);
    const nodeFields = fields(node, [spec.sequence, spec.goroutine, spec.processor_id, spec.callers, spec.next]);
    const gFields = fields(g, [spec.id, spec.parent_id, spec.status]);
    const stackFields = fields(stack, [module.layout.slice.data, module.layout.slice.length]);
    const frameFields = fields(frame, [spec.function, spec.file, spec.line]);
    const stringFields = Object.fromEntries([spec.function, spec.file].map(name =>
      [name, fields(fieldType(module, frame, name), [module.layout.string.data, module.layout.string.length])]));
    const strings = new Map();
    const text = async (bytes, base, name) => {
      const offset = base + frameFields[name].offset;
      const pointer = unsigned(bytes, stringFields[name][module.layout.string.data], offset);
      const length = unsigned(bytes, stringFields[name][module.layout.string.length], offset);
      if (length > BigInt(LIMIT) || (length !== 0n && pointer === 0n)) throw new Error(`invalid caller frame ${name}`);
      const key = `${pointer}:${length}`;
      if (!strings.has(key)) strings.set(key, new TextDecoder().decode(await memory(pointer, Number(length))));
      return strings.get(key);
    };
    const results = [], visited = new Set();
    while (nodeAddress !== 0n) {
      if (results.length >= LIMIT || visited.has(String(nodeAddress))) throw new Error('invalid or oversized goroutine registry');
      visited.add(String(nodeAddress));
      const nodeBytes = await memory(nodeAddress, node.size);
      const sequence = unsigned(nodeBytes, nodeFields[spec.sequence]);
      if (sequence & 1n) throw new Error('goroutine stack is changing; pause all workers outside runtime updates');
      const address = unsigned(nodeBytes, nodeFields[spec.goroutine]);
      if (address === 0n) throw new Error('goroutine registry contains a nil G');
      const gBytes = await memory(address, g.size);
      const id = unsigned(gBytes, gFields[spec.id]);
      const parent = unsigned(gBytes, gFields[spec.parent_id]);
      const state = unsigned(gBytes, gFields[spec.status]);
      const record = {id: String(id), parent: String(parent),
        state: module.layout.goroutine.status_names[String(state)] || `unknown(${state})`, processor: null, frames: []};
      const processor = BigInt.asIntN(nodeFields[spec.processor_id].size * 8,
        unsigned(nodeBytes, nodeFields[spec.processor_id]));
      if (processor >= 0n) record.processor = String(processor);
      const storeAddress = unsigned(nodeBytes, nodeFields[spec.callers]);
      if (storeAddress !== 0n) {
        const stackBytes = await memory(storeAddress + BigInt(stackField.offset), stack.size);
        const data = unsigned(stackBytes, stackFields[module.layout.slice.data]);
        const count = unsigned(stackBytes, stackFields[module.layout.slice.length]);
        if (count > BigInt(LIMIT) || (count !== 0n && data === 0n) || !Number.isInteger(frame.size) || frame.size <= 0) throw new Error('invalid or oversized logical stack');
        const frameBytes = await memory(data, Number(count) * frame.size);
        for (let index = Number(count) - 1; index >= 0; --index) {
          const base = index * frame.size;
          record.frames.push({function: await text(frameBytes, base, spec.function), file: await text(frameBytes, base, spec.file),
            line: String(unsigned(frameBytes, frameFields[spec.line], base))});
        }
      }
      // The trailing scalar reads must stay fresh, outside the cached blocks.
      if (await plugin.readNamedUnsigned(module, node, nodeAddress, spec.sequence, stopId) !== sequence) throw new Error('goroutine stack changed during inspection; pause all workers');
      results.push(record);
      nodeAddress = unsigned(nodeBytes, nodeFields[spec.next]);
    }
    if (await plugin.readNamedUnsigned(module, registry, located.value, spec.epoch, stopId) !== epoch) throw new Error('goroutine registry changed during inspection; pause all workers');
    return results;
  }

  globalThis.LLGoWasmGoroutines = {read};
})();
