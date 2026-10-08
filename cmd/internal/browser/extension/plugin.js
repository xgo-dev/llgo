// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

(() => {
  'use strict';

  const INDEX_CONTRACT = 'llgo.browser.debug';
  const INDEX_VERSION = 1;
  const RECORD_MAGIC = [0x4c, 0x4c, 0x47, 0x4f, 0x44, 0x42, 0x47, 0x00];
  const RECORD_SIZE = 16;
  const MAX_CHILDREN = 100;
  const MAX_CONTAINER_SCAN_BUCKETS = 4096;

  function readULEB(bytes, cursor) {
    let value = 0n;
    let shift = 0n;
    for (let count = 0; count < 10; ++count) {
      if (cursor.offset >= bytes.length) throw new Error('truncated varuint');
      const current = bytes[cursor.offset++];
      value |= BigInt(current & 0x7f) << shift;
      if ((current & 0x80) === 0) return value;
      shift += 7n;
    }
    throw new Error('invalid varuint');
  }

  function readSLEB(bytes, cursor) {
    let value = 0n;
    let shift = 0n;
    let current = 0;
    for (let count = 0; count < 10; ++count) {
      if (cursor.offset >= bytes.length) throw new Error('truncated varint');
      current = bytes[cursor.offset++];
      value |= BigInt(current & 0x7f) << shift;
      shift += 7n;
      if ((current & 0x80) === 0) {
        if ((current & 0x40) !== 0) value |= (-1n) << shift;
        return value;
      }
    }
    throw new Error('invalid varint');
  }

  function readName(bytes, cursor) {
    const size = Number(readULEB(bytes, cursor));
    if (cursor.offset + size > bytes.length) throw new Error('truncated WebAssembly name');
    const value = new TextDecoder().decode(bytes.subarray(cursor.offset, cursor.offset + size));
    cursor.offset += size;
    return value;
  }

  function customSections(moduleBytes) {
    const bytes = moduleBytes instanceof Uint8Array ? moduleBytes : new Uint8Array(moduleBytes);
    if (bytes.length < 8 || bytes[0] !== 0 || bytes[1] !== 0x61 || bytes[2] !== 0x73 || bytes[3] !== 0x6d) {
      throw new Error('invalid WebAssembly module');
    }
    const result = new Map();
    const cursor = {offset: 8};
    while (cursor.offset < bytes.length) {
      const id = bytes[cursor.offset++];
      const size = Number(readULEB(bytes, cursor));
      const end = cursor.offset + size;
      if (end > bytes.length) throw new Error('truncated WebAssembly section');
      if (id === 0) {
        const name = readName(bytes.subarray(0, end), cursor);
        if (result.has(name)) throw new Error(`multiple ${name} custom sections`);
        result.set(name, bytes.slice(cursor.offset, end));
      }
      cursor.offset = end;
    }
    return result;
  }

  function debuggerRecord(sections) {
    const bytes = sections.get('llgo.debugger');
    if (!bytes) return null;
    if (bytes.length !== RECORD_SIZE) throw new Error('invalid LLGo debugger record size');
    for (let index = 0; index < RECORD_MAGIC.length; ++index) {
      if (bytes[index] !== RECORD_MAGIC[index]) throw new Error('invalid LLGo debugger record magic');
    }
    if (bytes[12] !== 0 || bytes[15] !== 0) throw new Error('invalid LLGo debugger record reserved byte');
    return {
      record_version: bytes[8],
      schema_version: bytes[9],
      runtime_layout_version: bytes[10],
      llgo_abi_version: bytes[11],
      pointer_size: bytes[13],
      byte_order: bytes[14],
    };
  }

  function buildID(sections) {
    const bytes = sections.get('build_id');
    if (!bytes) return null;
    const cursor = {offset: 0};
    const size = Number(readULEB(bytes, cursor));
    if (size !== bytes.length - cursor.offset) throw new Error('invalid WebAssembly build_id');
    return [...bytes.subarray(cursor.offset)].map(value => value.toString(16).padStart(2, '0')).join('');
  }

  function inRanges(offset, ranges) {
    return !ranges || ranges.length === 0 || ranges.some(range =>
      ((!range.start && !range.end) || (offset >= range.start && offset < range.end)));
  }

  function bytesFromHex(value) {
    if (value.length % 2 !== 0) throw new Error('invalid hexadecimal DWARF expression');
    const result = new Uint8Array(value.length / 2);
    for (let index = 0; index < result.length; ++index) {
      result[index] = Number.parseInt(value.slice(index * 2, index * 2 + 2), 16);
    }
    return result;
  }

  function sameRecord(left, right) {
    return left.record_version === right.record_version &&
        left.schema_version === right.schema_version &&
        left.runtime_layout_version === right.runtime_layout_version &&
        left.llgo_abi_version === right.llgo_abi_version &&
        left.pointer_size === right.pointer_size && left.byte_order === right.byte_order;
  }

  class LLGoLanguageExtensionPlugin {
    constructor(languageServices, fetcher = globalThis.fetch.bind(globalThis)) {
      this.languageServices = languageServices;
      this.fetcher = fetcher;
      this.modules = new Map();
      this.objects = new Map();
      this.nextObject = 1;
      this.schema = null;
      this.addRawModuleCalls = 0;
      this.lastAddRawModuleError = null;
    }

    async addRawModule(rawModuleId, symbolsURL, rawModule) {
      ++this.addRawModuleCalls;
      try {
        return await this.loadRawModule(rawModuleId, symbolsURL, rawModule);
      } catch (error) {
        this.lastAddRawModuleError = String(error && error.stack || error);
        throw error;
      }
    }

    async loadRawModule(rawModuleId, symbolsURL, rawModule) {
      let code = rawModule.code;
      if (!code) {
        const response = await this.fetcher(rawModule.url);
        if (!response.ok) throw new Error(`load WebAssembly module: HTTP ${response.status}`);
        code = await response.arrayBuffer();
      }
      const sections = customSections(new Uint8Array(code));
      const record = debuggerRecord(sections);
      // A dedicated LLGo session must leave non-LLGo modules usable as raw
      // WebAssembly instead of claiming source presentation for them.
      if (!record) return [];
      const id = buildID(sections);
      if (!id) throw new Error('LLGo WebAssembly module has no build_id');

      const indexURL = new URL('/__llgo/debug-index.json', rawModule.url).href;
      const response = await this.fetcher(indexURL, {cache: 'no-store'});
      if (response.status === 424 || response.status === 404) {
        let missing = symbolsURL ? [symbolsURL] : [];
        try {
          const details = await response.json();
          if (Array.isArray(details.missing_symbol_files)) missing = details.missing_symbol_files;
        } catch (_) {
        }
        return {missingSymbolFiles: missing};
      }
      if (!response.ok) throw new Error(`load LLGo browser debug index: HTTP ${response.status}`);
      const index = await response.json();
      if (index.contract !== INDEX_CONTRACT || index.version !== INDEX_VERSION) {
        throw new Error(`unsupported LLGo browser debug index ${index.contract || '<missing>'} v${index.version}`);
      }
      if (index.build_id !== id) throw new Error('LLGo browser debug index build_id mismatch');
      if (!sameRecord(record, index.record)) throw new Error('LLGo browser debug index record mismatch');
      const schema = await this.loadSchema(rawModule.url);
      if (record.schema_version !== schema.schema_version ||
          record.runtime_layout_version !== schema.runtime_layout_version ||
          record.llgo_abi_version !== schema.llgo_abi_version) {
        throw new Error(`unsupported LLGo debugger schema/runtime/ABI ${record.schema_version}/${record.runtime_layout_version}/${record.llgo_abi_version}`);
      }

      const sources = new Map();
      const sourceByURL = new Map();
      for (const source of index.sources) {
        const resolved = {...source, resolvedURL: new URL(source.url, rawModule.url).href};
        sources.set(source.id, resolved);
        sourceByURL.set(resolved.resolvedURL, resolved);
      }
      const types = new Map(index.types.map(type => [type.id, type]));
      const typesByName = new Map();
      for (const type of index.types) {
        if (type.name && !typesByName.has(type.name)) typesByName.set(type.name, type);
      }
      const runtimeLayouts = schema.runtime_layouts || {};
      const layout = runtimeLayouts[String(record.runtime_layout_version)] || null;
      this.modules.set(rawModuleId, {
        rawModuleId, rawModule, symbolsURL, record, index, sources, sourceByURL, types, typesByName, layout,
      });
      const ready = await this.fetcher(new URL('/__llgo/plugin-ready', rawModule.url).href, {
        cache: 'no-store',
      });
      if (!ready.ok) throw new Error(`report LLGo browser debugger readiness: HTTP ${ready.status}`);
      return [...sources.values()].filter(source => source.local).map(source => source.resolvedURL);
    }

    async loadSchema(moduleURL) {
      if (!this.schema) {
        const schemaURL = new URL('/__llgo/debug-schema.json', moduleURL).href;
        const response = await this.fetcher(schemaURL, {cache: 'no-store'});
        if (!response.ok) throw new Error(`load LLGo debugger schema: HTTP ${response.status}`);
        this.schema = await response.json();
      }
      return this.schema;
    }

    async removeRawModule(rawModuleId) {
      this.modules.delete(rawModuleId);
      for (const [id, object] of this.objects) {
        if (object.rawModuleId === rawModuleId) this.objects.delete(id);
      }
    }

    module(rawModuleId) {
      const module = this.modules.get(rawModuleId);
      if (!module) throw new Error(`unknown LLGo raw module ${rawModuleId}`);
      return module;
    }

    async sourceLocationToRawLocation(location) {
      const module = this.module(location.rawModuleId);
      const source = module.sourceByURL.get(location.sourceFileURL);
      if (!source) return [];
      return module.index.lines
          .filter(line => line.source === source.id && line.line === location.lineNumber)
          .map(line => ({rawModuleId: location.rawModuleId, startOffset: line.start, endOffset: line.end}));
    }

    async rawLocationToSourceLocation(location) {
      const module = this.module(location.rawModuleId);
      return module.index.lines.filter(line => location.codeOffset >= line.start && location.codeOffset < line.end)
          .map(line => {
            const source = module.sources.get(line.source);
            return source ? {
              rawModuleId: location.rawModuleId,
              sourceFileURL: source.resolvedURL,
              lineNumber: line.line,
              columnNumber: line.column,
            } : null;
          }).filter(Boolean);
    }

    async getMappedLines(rawModuleId, sourceFileURL) {
      const module = this.module(rawModuleId);
      const source = module.sourceByURL.get(sourceFileURL);
      if (!source) return undefined;
      return [...new Set(module.index.lines.filter(line => line.source === source.id).map(line => line.line))]
          .sort((left, right) => left - right);
    }

    async getScopeInfo(type) {
      const names = {GLOBAL: 'Global', LOCAL: 'Local', PARAMETER: 'Parameter'};
      if (!names[type]) throw new Error(`unknown LLGo scope ${type}`);
      return {type, typeName: names[type], icon: 'data:null'};
    }

    activeVariables(module, codeOffset) {
      const active = module.index.variables.filter(variable => inRanges(codeOffset, variable.ranges) &&
          (variable.constant || variable.locations.some(location => inRanges(codeOffset, [location]))));
      active.sort((left, right) => right.depth - left.depth);
      const names = new Set();
      return active.filter(variable => {
        if (names.has(variable.name)) return false;
        names.add(variable.name);
        return true;
      });
    }

    async listVariablesInScope(location) {
      const module = this.module(location.rawModuleId);
      const result = this.activeVariables(module, location.codeOffset).map(variable => ({
        scope: variable.scope,
        name: variable.name,
        type: module.types.get(variable.type)?.name || '<unknown>',
      }));
      return result;
    }

    async getFunctionInfo(location) {
      const module = this.module(location.rawModuleId);
      const matches = module.index.functions.filter(fn => inRanges(location.codeOffset, fn.ranges));
      matches.sort((left, right) => rangeWidth(left.ranges) - rangeWidth(right.ranges));
      return {frames: matches.map(fn => ({name: fn.name})), missingSymbolFiles: []};
    }

    async getInlinedFunctionRanges() { return []; }
    async getInlinedCalleesRanges() { return []; }

    async evaluate(expression, context, stopId) {
      const trimmed = expression.trim();
      if (!trimmed) return null;
      const module = this.module(context.rawModuleId);
      if (trimmed === '$goroutines') {
        const records = await globalThis.LLGoWasmGoroutines.read(this, module, context, stopId);
        if (records === null) {
          return {type: 'string', value: 'goroutine inspection unavailable: build with llgo.wasm.debugger', hasChildren: false};
        }
        const properties = records.map(record => ({name: String(record.id), value: {
          type: 'object', className: 'Go goroutine',
          description: `goroutine ${record.id} [${record.state}] P${record.processor ?? '<unattached>'}`,
          properties: [
          {name: 'id', value: {type: 'string', value: record.id, hasChildren: false}},
          {name: 'parent', value: {type: 'string', value: record.parent, hasChildren: false}},
          {name: 'state', value: {type: 'string', value: record.state, hasChildren: false}},
          {name: 'processor', value: {type: 'string', value: record.processor ?? '<unattached>', hasChildren: false}},
          {name: 'frames', value: {type: 'array', description: `${record.frames.length} logical frames`,
            properties: record.frames.map((frame, index) => ({name: String(index), value: {
              type: 'string', value: `${frame.function} (${frame.file}:${frame.line})`, hasChildren: false,
            }}))}},
        ]}}));
        return this.snapshotObject(module, {type: 'array', description: `${records.length} Go goroutines`}, properties);
      }
      const variables = this.activeVariables(module, context.codeOffset);
      const variable = variables.filter(item => trimmed === item.name || trimmed.startsWith(item.name + '.'))
          .sort((left, right) => right.name.length - left.name.length)[0];
      if (!variable) return null;
      const suffix = trimmed.slice(variable.name.length);
      const path = suffix ? suffix.slice(1).split('.').filter(Boolean) : [];
      let type = module.types.get(variable.type);
      if (!type) return null;
      let located = await this.variableLocation(variable, context.codeOffset, module, stopId);
      if (!located) return null;
      for (const fieldName of path) {
        const resolved = resolveType(module, type);
        if (located.kind !== 'address' || !resolved.fields) return null;
        const field = resolved.fields.find(item => item.name === fieldName);
        if (!field) return null;
        located = {kind: 'address', value: located.value + BigInt(field.offset)};
        type = module.types.get(field.type);
        if (!type) return null;
      }
      return this.remoteObject(module, type, located, stopId);
    }

    async variableLocation(variable, codeOffset, module, stopId) {
      if (variable.constant) {
        const value = constantToValue(variable.constant);
        return value === null ? null : {kind: 'value', value};
      }
      const location = variable.locations.find(item => inRanges(codeOffset, [item]));
      if (!location) return null;
      const frameBase = variable.frame_base?.find(item => inRanges(codeOffset, [item]));
      return this.evaluateDWARF(bytesFromHex(location.expression), module, stopId, frameBase?.expression);
    }

    async evaluateDWARF(bytes, module, stopId, frameBase) {
      const cursor = {offset: 0};
      const stack = [];
      let stackValue = false;
      const pop = () => {
        if (!stack.length) throw new Error('invalid empty DWARF expression stack');
        return stack.pop();
      };
      while (cursor.offset < bytes.length) {
        const op = bytes[cursor.offset++];
        if (op >= 0x30 && op <= 0x4f) {
          stack.push(BigInt(op - 0x30));
          continue;
        }
        switch (op) {
          case 0x03: stack.push(readFixed(bytes, cursor, module.record.pointer_size, false)); break; // DW_OP_addr
          case 0x06: {
            const address = pop();
            stack.push(await this.readUnsigned(address, module.record.pointer_size, stopId));
            break;
          }
          case 0x08: stack.push(readFixed(bytes, cursor, 1, false)); break;
          case 0x09: stack.push(readFixed(bytes, cursor, 1, true)); break;
          case 0x0a: stack.push(readFixed(bytes, cursor, 2, false)); break;
          case 0x0b: stack.push(readFixed(bytes, cursor, 2, true)); break;
          case 0x0c: stack.push(readFixed(bytes, cursor, 4, false)); break;
          case 0x0d: stack.push(readFixed(bytes, cursor, 4, true)); break;
          case 0x0e: stack.push(readFixed(bytes, cursor, 8, false)); break;
          case 0x0f: stack.push(readFixed(bytes, cursor, 8, true)); break;
          case 0x10: stack.push(readULEB(bytes, cursor)); break;
          case 0x11: stack.push(readSLEB(bytes, cursor)); break;
          case 0x12: stack.push(stack[stack.length - 1]); break;
          case 0x13: pop(); break;
          case 0x14: stack.push(stack[stack.length - 2]); break;
          case 0x1a: { const right = pop(); stack.push(pop() & right); break; }
          case 0x1b: { const right = pop(); stack.push(pop() / right); break; }
          case 0x1c: { const right = pop(); stack.push(pop() - right); break; }
          case 0x1d: { const right = pop(); stack.push(pop() % right); break; }
          case 0x1e: { const right = pop(); stack.push(pop() * right); break; }
          case 0x21: { const right = pop(); stack.push(pop() | right); break; }
          case 0x22: { const right = pop(); stack.push(pop() + right); break; }
          case 0x23: stack.push(pop() + readULEB(bytes, cursor)); break;
          case 0x24: { const right = pop(); stack.push(pop() << right); break; }
          case 0x25: { const right = pop(); stack.push(pop() >> right); break; }
          case 0x27: { const right = pop(); stack.push(pop() ^ right); break; }
          case 0x91: { // DW_OP_fbreg: locations of O0 locals and address-backed parameters.
            if (!frameBase) return null;
            const base = await this.evaluateDWARF(bytesFromHex(frameBase), module, stopId);
            if (!base) return null;
            stack.push(base.value + readSLEB(bytes, cursor));
            break;
          }
          case 0x9f: stackValue = true; break;
          case 0xed: {
            if (cursor.offset >= bytes.length) throw new Error('truncated DW_OP_WASM_location');
            const kind = bytes[cursor.offset++];
            const index = kind === 3 ? Number(readFixed(bytes, cursor, 4, false)) : Number(readULEB(bytes, cursor));
            let wasm;
            if (kind === 0) wasm = await this.languageServices.getWasmLocal(index, stopId);
            else if (kind === 1 || kind === 3) wasm = await this.languageServices.getWasmGlobal(index, stopId);
            else if (kind === 2) wasm = await this.languageServices.getWasmOp(index, stopId);
            else throw new Error(`unsupported DW_OP_WASM_location kind ${kind}`);
            if (wasm.type === 'reftype') return null;
            stack.push(wasm.type === 'f32' || wasm.type === 'f64' ? wasm.value : BigInt(wasm.value));
            break;
          }
          default:
            throw new Error(`unsupported DWARF expression opcode 0x${op.toString(16)}`);
        }
      }
      if (stack.length !== 1) throw new Error('invalid DWARF expression result');
      return {kind: stackValue ? 'value' : 'address', value: stack[0]};
    }

    async remoteObject(module, originalType, located, stopId) {
      const type = resolveType(module, originalType);
      if (!type) return null;
      if (located.kind === 'value' && isScalar(type)) return scalarRemote(type, located.value);
      const address = located.value;
      if (isScalar(type)) {
        const value = await this.readScalar(type, address, stopId);
        return scalarRemote(type, value);
      }
      const stringSpec = module.layout?.string;
      if (stringSpec && runtimeTypeMatches(originalType, type, stringSpec.type_name)) {
        return this.stringRemote(module, type, address, stopId, stringSpec);
      }
      const sliceSpec = module.layout?.slice;
      if (sliceSpec && runtimeTypeMatches(originalType, type, sliceSpec.type_pattern, true)) {
        return this.sliceRemote(module, type, address, stopId, sliceSpec);
      }
      const interfaceSpec = module.layout?.interface;
      if (interfaceSpec && runtimeTypeMatches(originalType, type, interfaceSpec.type_pattern, true)) {
        return this.interfaceRemote(module, originalType, type, address, stopId, interfaceSpec);
      }
      const functionSpec = module.layout?.function;
      if (functionSpec && runtimeTypeMatches(originalType, type, functionSpec.type_pattern, true)) {
        return this.functionRemote(module, originalType, type, address, stopId, functionSpec);
      }
      const mapSpec = module.layout?.map;
      if (mapSpec && (runtimeTypeMatches(originalType, type, mapSpec.type_pattern, true) ||
          resolvePointee(module, type)?.name?.startsWith('hash<'))) {
        return this.mapRemote(module, originalType, type, located, stopId, mapSpec);
      }
      const channelSpec = module.layout?.channel;
      if (channelSpec && (runtimeTypeMatches(originalType, type, channelSpec.type_pattern, true) ||
          resolvePointee(module, type)?.name?.startsWith('hchan<'))) {
        return this.channelRemote(module, originalType, type, located, stopId, channelSpec);
      }
      if (type.kind === 'pointer') {
        const pointer = await this.runtimePointerValue(module, type, located, stopId);
        const object = this.storeObject(module, type, pointer, stopId, 'pointer');
        return {
          type: 'object', className: originalType.name || type.name,
          description: pointer === 0n ? 'nil' : `0x${pointer.toString(16)}`,
          objectId: object, hasChildren: pointer !== 0n,
          linearMemoryAddress: numberAddress(pointer), linearMemorySize: 0,
        };
      }
      const objectId = this.storeObject(module, type, address, stopId, 'aggregate');
      return {
        type: type.kind === 'array' ? 'array' : 'object', className: originalType.name || type.name,
        description: originalType.name || type.name, objectId, hasChildren: true,
        linearMemoryAddress: numberAddress(address), linearMemorySize: Math.max(0, type.size),
      };
    }

    async stringRemote(module, type, address, stopId, spec) {
      const dataField = type.fields.find(field => field.name === spec.data);
      const lengthField = type.fields.find(field => field.name === spec.length);
      if (!dataField || !lengthField) return null;
      const pointer = await this.readUnsigned(address + BigInt(dataField.offset), module.record.pointer_size, stopId);
      const length = await this.readUnsigned(address + BigInt(lengthField.offset), module.record.pointer_size, stopId);
      const size = Number(length > 4096n ? 4096n : length);
      const raw = size ? await this.languageServices.getWasmLinearMemory(numberAddress(pointer), size, stopId) : new ArrayBuffer(0);
      const text = new TextDecoder().decode(raw);
      const truncated = length > 4096n ? '…' : '';
      return {
        type: 'string', className: type.name, value: text,
        description: JSON.stringify(text + truncated), hasChildren: false,
        linearMemoryAddress: numberAddress(pointer), linearMemorySize: Number(length),
      };
    }

    async sliceRemote(module, type, address, stopId, spec) {
      const dataField = type.fields.find(field => field.name === spec.data);
      const lengthField = type.fields.find(field => field.name === spec.length);
      const capacityField = type.fields.find(field => field.name === spec.capacity);
      if (!dataField || !lengthField || !capacityField) return null;
      const pointer = await this.readUnsigned(address + BigInt(dataField.offset), module.record.pointer_size, stopId);
      const length = await this.readUnsigned(address + BigInt(lengthField.offset), module.record.pointer_size, stopId);
      const capacity = await this.readUnsigned(address + BigInt(capacityField.offset), module.record.pointer_size, stopId);
      const objectId = this.storeObject(module, type, address, stopId, 'slice', {pointer, length});
      return {
        type: 'array', className: type.name, description: `${type.name} len=${length} cap=${capacity}`,
        objectId, hasChildren: length !== 0n,
        linearMemoryAddress: numberAddress(pointer), linearMemorySize: 0,
      };
    }

    async interfaceRemote(module, originalType, type, address, stopId, spec) {
      const typeField = fieldByName(type, spec.type);
      const dataField = fieldByName(type, spec.data);
      if (!typeField || !dataField) return null;
      let typePointer = await this.readUnsigned(
          address + BigInt(typeField.offset), module.record.pointer_size, stopId);
      const dataPointer = await this.readUnsigned(
          address + BigInt(dataField.offset), module.record.pointer_size, stopId);
      if (typePointer === 0n) {
        return {type: 'null', value: null, description: 'nil', hasChildren: false};
      }
      const displayType = originalType.name || type.name;
      if (type.name !== spec.empty_type) {
        const itabType = lookupType(module, spec.itab_type);
        const concreteField = fieldByName(resolveType(module, itabType), spec.itab_concrete_type);
        if (!concreteField) return null;
        typePointer = await this.readUnsigned(
            typePointer + BigInt(concreteField.offset), module.record.pointer_size, stopId);
        if (typePointer === 0n) return null;
      }
      const dynamicName = await this.runtimeTypeName(module, typePointer, stopId);
      const dynamicType = dynamicName ? lookupType(module, dynamicName) : null;
      const runtimeSpec = module.layout?.runtime_type;
      const runtimeType = resolveType(module, lookupType(module, runtimeSpec?.type_name));
      const kind = runtimeSpec?.kind ? await this.readNamedUnsigned(
          module, runtimeType, typePointer, runtimeSpec.kind, stopId) : null;
      // Direct interfaces store their value in the data word itself. Using the
      // address of that word also handles pointer-shaped structs and typed nils
      // without adding another indirection or pretending the interface is nil.
      const direct = kind !== null && runtimeSpec.direct_interface_flag !== undefined ?
          (kind & BigInt(runtimeSpec.direct_interface_flag)) !== 0n : null;
      const valueAddress = direct === null ? null :
          direct ? address + BigInt(dataField.offset) : dataPointer;
      const objectId = this.storeObject(module, type, address, stopId, 'interface', {
        valueAddress, dynamicType,
      });
      return {
        type: 'object', className: displayType,
        description: `type=${dynamicName || `0x${typePointer.toString(16)}`}`,
        objectId, hasChildren: !!dynamicType && valueAddress !== null && valueAddress !== 0n,
        linearMemoryAddress: numberAddress(address), linearMemorySize: Math.max(0, type.size),
      };
    }

    async runtimeTypeName(module, address, stopId) {
      if (!address) return null;
      const spec = module.layout?.runtime_type;
      if (!spec) return null;
      const runtimeType = resolveType(module, lookupType(module, spec.type_name));
      const stringField = fieldByName(runtimeType, spec.string);
      if (!runtimeType || !stringField) return null;
      const stringType = resolveType(module, module.types.get(stringField.type));
      const name = await this.readGoString(module, stringType, address + BigInt(stringField.offset), stopId, 4096);
      if (name === null) return null;
      const flagField = fieldByName(runtimeType, spec.tflag);
      if (!flagField) return name;
      const flagType = resolveType(module, module.types.get(flagField.type));
      const flags = await this.readUnsigned(
          address + BigInt(flagField.offset), Math.max(1, flagType?.size || 1), stopId);
      return (flags & BigInt(spec.extra_star_flag)) !== 0n ? `*${name}` : name;
    }

    async readGoString(module, type, address, stopId, limit) {
      const spec = module.layout?.string;
      if (!spec) return null;
      const pointer = await this.readNamedUnsigned(module, type, address, spec.data, stopId);
      const length = await this.readNamedUnsigned(module, type, address, spec.length, stopId);
      if (pointer === null || length === null) return null;
      if (length > BigInt(limit) || (length !== 0n && pointer === 0n)) return null;
      const raw = length === 0n ? new ArrayBuffer(0) : await this.languageServices.getWasmLinearMemory(
          numberAddress(pointer), Number(length), stopId);
      return new TextDecoder().decode(raw);
    }

    async functionRemote(module, originalType, type, address, stopId, spec) {
      const codeField = fieldByName(type, spec.code);
      const dataField = fieldByName(type, spec.data);
      if (!codeField || !dataField) return null;
      const code = await this.readUnsigned(
          address + BigInt(codeField.offset), module.record.pointer_size, stopId);
      const data = await this.readUnsigned(
          address + BigInt(dataField.offset), module.record.pointer_size, stopId);
      if (code === 0n) return {type: 'null', value: null, description: 'nil', hasChildren: false};
      // A Wasm function pointer is a table index, not a code-section offset.
      // Resolving it needs the table/element mapping, which the index does not
      // yet contain. Keep the index explicit instead of guessing a DWARF name.
      let name = `func[${code}]`;
      if (data !== 0n) name += ` data=0x${data.toString(16)}`;
      const objectId = this.storeObject(module, type, address, stopId, 'aggregate');
      return {
        type: 'object', className: originalType.name || type.name, description: name,
        objectId, hasChildren: true,
        linearMemoryAddress: numberAddress(address), linearMemorySize: Math.max(0, type.size),
      };
    }

    async mapRemote(module, originalType, type, located, stopId, spec) {
      const pointer = await this.runtimePointerValue(module, type, located, stopId);
      if (pointer === null) return null;
      if (pointer === 0n) return {type: 'null', value: null, description: 'nil', hasChildren: false};
      const hashType = resolvePointee(module, type);
      const count = await this.readNamedUnsigned(module, hashType, pointer, spec.count, stopId);
      if (count === null) return null;
      const objectId = this.storeObject(module, type, pointer, stopId, 'map', {
        hashType, length: count, spec,
      });
      return {
        type: 'object', className: originalType.name || type.name, description: `len=${count}`,
        objectId, hasChildren: count !== 0n,
        linearMemoryAddress: numberAddress(pointer), linearMemorySize: Math.max(0, hashType?.size || 0),
      };
    }

    async channelRemote(module, originalType, type, located, stopId, spec) {
      const pointer = await this.runtimePointerValue(module, type, located, stopId);
      if (pointer === null) return null;
      if (pointer === 0n) return {type: 'null', value: null, description: 'nil', hasChildren: false};
      const channelType = resolvePointee(module, type);
      const length = await this.readNamedUnsigned(module, channelType, pointer, spec.count, stopId);
      const capacity = await this.readNamedUnsigned(module, channelType, pointer, spec.capacity, stopId);
      const buffer = await this.readNamedUnsigned(module, channelType, pointer, spec.buffer, stopId);
      const receiveIndex = await this.readNamedUnsigned(module, channelType, pointer, spec.receive_index, stopId);
      const closed = await this.readNamedUnsigned(module, channelType, pointer, spec.closed, stopId);
      if ([length, capacity, buffer, receiveIndex, closed].some(value => value === null)) return null;
      const elementType = channelElementType(module, channelType, spec);
      const objectId = this.storeObject(module, type, pointer, stopId, 'channel', {
        length, capacity, buffer, receiveIndex, elementType,
      });
      return {
        type: 'array', className: originalType.name || type.name,
        description: `len=${length} cap=${capacity}${closed !== 0n ? ' closed' : ''}`,
        objectId, hasChildren: length !== 0n && buffer !== 0n && !!elementType,
        linearMemoryAddress: numberAddress(buffer), linearMemorySize: 0,
      };
    }

    async runtimePointerValue(module, type, located, stopId) {
      if (located.kind === 'value') {
        return BigInt.asUintN(module.record.pointer_size * 8, BigInt(located.value));
      }
      return type.kind === 'pointer' ?
        this.readUnsigned(located.value, Math.max(1, type.size || module.record.pointer_size), stopId) : located.value;
    }

    async readNamedUnsigned(module, type, address, name, stopId) {
      const field = fieldByName(type, name);
      if (!field) return null;
      const fieldType = resolveType(module, module.types.get(field.type));
      const size = fieldType?.kind === 'pointer' ? module.record.pointer_size : fieldType?.size;
      if (!size || size < 1 || size > 8) return null;
      return this.readUnsigned(address + BigInt(field.offset), size, stopId);
    }

    storeObject(module, type, address, stopId, kind, extra = {}) {
      const id = `llgo:${this.nextObject++}`;
      this.objects.set(id, {rawModuleId: module.rawModuleId, type, address, stopId, kind, ...extra});
      return id;
    }

    snapshotObject(module, summary, properties) {
      const objectId = this.storeObject(module, null, null, null, 'snapshot', {properties});
      return {...summary, objectId, hasChildren: properties.length !== 0};
    }

    async getProperties(objectId) {
      const object = this.objects.get(objectId);
      if (!object) return [];
      const module = this.module(object.rawModuleId);
      if (object.kind === 'snapshot') {
        if (!object.resolvedProperties) {
          object.children = [];
          object.resolvedProperties = object.properties.map(property => {
            if (!property.value.properties) return property;
            const {properties, ...summary} = property.value;
            const value = this.snapshotObject(module, summary, properties);
            object.children.push(value.objectId);
            return {...property, value};
          });
        }
        return object.resolvedProperties;
      }
      const type = resolveType(module, object.type);
      if (object.kind === 'pointer') {
        if (object.address === 0n) return [];
        const elem = module.types.get(type.elem);
        return [{name: '*', value: await this.remoteObject(module, elem, {kind: 'address', value: object.address}, object.stopId)}];
      }
      if (object.kind === 'slice') {
        const dataField = type.fields.find(field => field.name === module.layout.slice.data);
        const pointerType = dataField ? resolveType(module, module.types.get(dataField.type)) : null;
        const elem = pointerType?.elem ? module.types.get(pointerType.elem) : null;
        if (!elem || elem.size <= 0) return [];
        const count = Number(object.length > BigInt(MAX_CHILDREN) ? BigInt(MAX_CHILDREN) : object.length);
        const result = [];
        for (let index = 0; index < count; ++index) {
          result.push({
            name: String(index),
            value: await this.remoteObject(module, elem, {
              kind: 'address', value: object.pointer + BigInt(index) * BigInt(elem.size),
            }, object.stopId),
          });
        }
        return result;
      }
      if (object.kind === 'interface') {
        if (!object.dynamicType || object.valueAddress === null || object.valueAddress === 0n) return [];
        return [{name: 'value', value: await this.remoteObject(
          module, object.dynamicType, {kind: 'address', value: object.valueAddress}, object.stopId)}];
      }
      if (object.kind === 'map') return this.mapProperties(module, object);
      if (object.kind === 'channel') return this.channelProperties(module, object);
      if (type.kind === 'array') {
        const elem = module.types.get(type.elem);
        if (!elem || elem.size <= 0) return [];
        const count = Math.min(type.count, MAX_CHILDREN);
        const result = [];
        for (let index = 0; index < count; ++index) {
          result.push({name: String(index), value: await this.remoteObject(module, elem, {
            kind: 'address', value: object.address + BigInt(index) * BigInt(elem.size),
          }, object.stopId)});
        }
        return result;
      }
      const result = [];
      for (const field of type.fields || []) {
        const fieldType = module.types.get(field.type);
        if (!fieldType) continue;
        result.push({name: field.name, value: await this.remoteObject(module, fieldType, {
          kind: 'address', value: object.address + BigInt(field.offset),
        }, object.stopId)});
      }
      return result;
    }

    async channelProperties(module, object) {
      const {length, capacity, buffer, receiveIndex, elementType} = object;
      if (!elementType || elementType.size <= 0 || capacity === 0n || buffer === 0n) return [];
      const count = Number(length > BigInt(MAX_CHILDREN) ? BigInt(MAX_CHILDREN) : length);
      const result = [];
      for (let index = 0; index < count; ++index) {
        const slot = (receiveIndex + BigInt(index)) % capacity;
        result.push({name: String(index), value: await this.remoteObject(module, elementType, {
          kind: 'address', value: buffer + slot * BigInt(elementType.size),
        }, object.stopId)});
      }
      return result;
    }

    async mapProperties(module, object) {
      const {hashType, spec, length} = object;
      if (!hashType || length === 0n) return [];
      const flags = await this.readNamedUnsigned(module, hashType, object.address, spec.flags, object.stopId);
      const bits = await this.readNamedUnsigned(module, hashType, object.address, spec.bucket_bits, object.stopId);
      const buckets = await this.readNamedUnsigned(module, hashType, object.address, spec.buckets, object.stopId);
      const oldBuckets = await this.readNamedUnsigned(module, hashType, object.address, spec.old_buckets, object.stopId);
      if ([flags, bits, buckets, oldBuckets].some(value => value === null) || bits >= 63n || buckets === 0n) return [];
      const bucketsField = fieldByName(hashType, spec.buckets);
      const bucketsPointer = resolveType(module, module.types.get(bucketsField?.type));
      const bucketType = resolvePointee(module, bucketsPointer);
      if (!bucketType || bucketType.size <= 0) return [];
      const logical = 1n << bits;
      const scan = Number(logical > BigInt(MAX_CONTAINER_SCAN_BUCKETS) ?
        BigInt(MAX_CONTAINER_SCAN_BUCKETS) : logical);
      const oldCount = (flags & BigInt(spec.same_size_grow_flag)) !== 0n ? logical : logical >> 1n;
      const result = [];
      for (let bucketIndex = 0; bucketIndex < scan && result.length < MAX_CHILDREN * 2; ++bucketIndex) {
        let bucketAddress = buckets + BigInt(bucketIndex) * BigInt(bucketType.size);
        if (oldBuckets !== 0n && oldCount !== 0n) {
          const oldIndex = BigInt(bucketIndex) & (oldCount - 1n);
          const oldAddress = oldBuckets + oldIndex * BigInt(bucketType.size);
          if (!(await this.bucketEvacuated(module, bucketType, oldAddress, object.stopId, spec))) {
            if (BigInt(bucketIndex) >= oldCount) continue;
            bucketAddress = oldAddress;
          }
        }
        const visited = new Set();
        while (bucketAddress !== 0n && !visited.has(bucketAddress.toString()) &&
               result.length < MAX_CHILDREN * 2) {
          visited.add(bucketAddress.toString());
          const entries = await this.bucketEntries(module, bucketType, bucketAddress, object.stopId, spec);
          if (!entries) return result;
          for (const entry of entries) {
            const index = result.length / 2;
            result.push({name: `key[${index}]`, value: await this.remoteObject(
              module, entry.keyType, {kind: 'address', value: entry.key}, object.stopId)});
            result.push({name: `value[${index}]`, value: await this.remoteObject(
              module, entry.valueType, {kind: 'address', value: entry.value}, object.stopId)});
            if (result.length >= MAX_CHILDREN * 2 || BigInt(result.length / 2) >= length) return result;
          }
          bucketAddress = await this.readNamedUnsigned(
              module, bucketType, bucketAddress, spec.bucket_overflow, object.stopId) || 0n;
        }
      }
      return result;
    }

    async bucketEvacuated(module, bucketType, address, stopId, spec) {
      const field = fieldByName(bucketType, spec.bucket_tophash);
      const array = resolveType(module, module.types.get(field?.type));
      const element = array?.kind === 'array' ? resolveType(module, module.types.get(array.elem)) : null;
      if (!field || !element || element.size <= 0) return false;
      const value = await this.readUnsigned(address + BigInt(field.offset), element.size, stopId);
      return value >= BigInt(spec.evacuated_tophash_min) && value <= BigInt(spec.evacuated_tophash_max);
    }

    async bucketEntries(module, bucketType, address, stopId, spec) {
      const topField = fieldByName(bucketType, spec.bucket_tophash);
      const keyField = fieldByName(bucketType, spec.bucket_keys) ||
          fieldByName(bucketType, spec.bucket_indirect_keys);
      const valueField = fieldByName(bucketType, spec.bucket_values) ||
          fieldByName(bucketType, spec.bucket_indirect_values);
      if (!topField || !keyField || !valueField) return null;
      const topArray = resolveType(module, module.types.get(topField.type));
      const keyArray = resolveType(module, module.types.get(keyField.type));
      const valueArray = resolveType(module, module.types.get(valueField.type));
      if (topArray?.kind !== 'array' || keyArray?.kind !== 'array' || valueArray?.kind !== 'array') return null;
      const topType = resolveType(module, module.types.get(topArray.elem));
      const keyStorageType = resolveType(module, module.types.get(keyArray.elem));
      const valueStorageType = resolveType(module, module.types.get(valueArray.elem));
      if (!topType || !keyStorageType || !valueStorageType || topType.size <= 0 ||
          keyStorageType.size <= 0 || valueStorageType.size <= 0) return null;
      const indirectKey = keyField.name === spec.bucket_indirect_keys;
      const indirectValue = valueField.name === spec.bucket_indirect_values;
      const keyType = indirectKey ? resolvePointee(module, keyStorageType) : keyStorageType;
      const valueType = indirectValue ? resolvePointee(module, valueStorageType) : valueStorageType;
      if (!keyType || !valueType) return null;
      const slots = Math.min(topArray.count, keyArray.count, valueArray.count);
      const result = [];
      for (let slot = 0; slot < slots; ++slot) {
        const top = await this.readUnsigned(
            address + BigInt(topField.offset + slot * topType.size), topType.size, stopId);
        if (top < BigInt(spec.occupied_tophash_min)) continue;
        let key = address + BigInt(keyField.offset + slot * keyStorageType.size);
        let value = address + BigInt(valueField.offset + slot * valueStorageType.size);
        if (indirectKey) {
          key = await this.readUnsigned(key, module.record.pointer_size, stopId);
        }
        if (indirectValue) {
          value = await this.readUnsigned(value, module.record.pointer_size, stopId);
        }
        if (key !== 0n && value !== 0n && keyType && valueType) result.push({key, value, keyType, valueType});
      }
      return result;
    }

    async releaseObject(objectId) {
      const object = this.objects.get(objectId);
      this.objects.delete(objectId);
      for (const child of object?.children || []) await this.releaseObject(child);
    }

    async readUnsigned(address, size, stopId) {
      const raw = await this.languageServices.getWasmLinearMemory(numberAddress(address), size, stopId);
      const bytes = new Uint8Array(raw);
      let result = 0n;
      for (let index = 0; index < bytes.length; ++index) result |= BigInt(bytes[index]) << BigInt(index * 8);
      return result;
    }

    async readScalar(type, address, stopId) {
      const raw = await this.languageServices.getWasmLinearMemory(numberAddress(address), type.size, stopId);
      const view = new DataView(raw);
      if (type.kind === 'float') return type.size === 4 ? view.getFloat32(0, true) : view.getFloat64(0, true);
      let value = 0n;
      const bytes = new Uint8Array(raw);
      for (let index = 0; index < bytes.length; ++index) value |= BigInt(bytes[index]) << BigInt(index * 8);
      return signedValue(type, value);
    }
  }

  function runtimeTypeMatches(originalType, resolvedType, pattern, regexp = false) {
    if (!pattern) return false;
    const names = new Set([originalType?.name, resolvedType?.name].filter(Boolean));
    if (!regexp) return names.has(pattern);
    const expression = new RegExp(pattern);
    return [...names].some(name => expression.test(name));
  }

  function lookupType(module, name) {
    if (!name) return null;
    return module.typesByName.get(name) || module.typesByName.get(`struct ${name}`) || null;
  }

  function resolvePointee(module, type) {
    type = resolveType(module, type);
    return type?.kind === 'pointer' ? resolveType(module, module.types.get(type.elem)) : null;
  }

  function fieldByName(type, name) {
    return name && type?.fields ? type.fields.find(field => field.name === name) || null : null;
  }

  function channelElementType(module, channelType, spec) {
    const queueField = fieldByName(channelType, spec.receive_queue);
    const queueType = resolveType(module, module.types.get(queueField?.type));
    const firstField = fieldByName(queueType, spec.queue_first);
    const waiterPointer = resolveType(module, module.types.get(firstField?.type));
    const waiterType = resolvePointee(module, waiterPointer);
    const elementField = fieldByName(waiterType, spec.waiter_element);
    const elementPointer = resolveType(module, module.types.get(elementField?.type));
    return resolvePointee(module, elementPointer);
  }

  function resolveType(module, type) {
    const seen = new Set();
    while (type && (type.kind === 'typedef' || type.kind === 'qualified')) {
      if (seen.has(type.id)) return null;
      seen.add(type.id);
      type = module.types.get(type.elem);
    }
    return type;
  }

  function isScalar(type) {
    return ['bool', 'integer', 'float', 'enum'].includes(type.kind);
  }

  function signedValue(type, value) {
    if (type.size <= 0) return value;
    // Wasm locals can already be signed and readScalar may have normalized a
    // memory value. Width conversion must be idempotent for both paths.
    const bits = type.size * 8;
    return type.signed ? BigInt.asIntN(bits, value) : BigInt.asUintN(bits, value);
  }

  function scalarRemote(type, raw) {
    if (type.kind === 'bool') {
      const value = raw !== 0n && raw !== 0;
      return {type: 'boolean', value, description: String(value), hasChildren: false};
    }
    if (typeof raw === 'number') {
      return {type: 'number', value: raw, description: String(raw), hasChildren: false};
    }
    const value = signedValue(type, raw);
    if (value <= BigInt(Number.MAX_SAFE_INTEGER) && value >= BigInt(Number.MIN_SAFE_INTEGER)) {
      return {type: 'number', value: Number(value), description: value.toString(), hasChildren: false};
    }
    return {type: 'bigint', value, description: `${value}n`, hasChildren: false};
  }

  function constantToValue(constant) {
    if (constant.kind === 'signed' || constant.kind === 'unsigned') return BigInt(constant.value || '0');
    // DW_AT_const_value can also encode strings and raw bytes. Until those
    // encodings are interpreted using the variable type, report unavailable.
    return null;
  }

  function readFixed(bytes, cursor, size, signed) {
    if (cursor.offset + size > bytes.length) throw new Error('truncated fixed-width DWARF operand');
    let result = 0n;
    for (let index = 0; index < size; ++index) result |= BigInt(bytes[cursor.offset++]) << BigInt(index * 8);
    if (signed) {
      const bits = BigInt(size * 8);
      const sign = 1n << (bits - 1n);
      if ((result & sign) !== 0n) result -= 1n << bits;
    }
    return result;
  }

  function numberAddress(value) {
    if (value < 0n || value > BigInt(Number.MAX_SAFE_INTEGER)) throw new Error(`invalid linear-memory address ${value}`);
    return Number(value);
  }

  function rangeWidth(ranges) {
    return (ranges || []).reduce((total, range) => total + Math.max(0, range.end - range.start), 0);
  }

  globalThis.LLGoLanguageExtension = {
    LLGoLanguageExtensionPlugin,
    customSections,
    debuggerRecord,
    buildID,
    bytesFromHex,
  };

  if (globalThis.chrome?.devtools?.languageServices) {
    const languageServices = globalThis.chrome.devtools.languageServices;
    const plugin = new LLGoLanguageExtensionPlugin(languageServices);
    globalThis.__llgoLanguageExtensionPlugin = plugin;
    globalThis.__llgoLanguageExtensionRegistration = languageServices.registerLanguageExtensionPlugin(
        plugin, 'LLGo WebAssembly Debugger',
        {language: 'WebAssembly', symbol_types: ['EmbeddedDWARF', 'ExternalDWARF']})
        .then(() => new Promise(resolve => {
          // A Wasm module that is instantiated before DevTools has installed the
          // language plugin may not be offered to the plugin. Tell the inspected
          // LLGo launcher that registration is complete before it instantiates.
          chrome.devtools.inspectedWindow.eval(
              `globalThis.__llgoLanguageExtensionReady = true;
               globalThis.dispatchEvent(new Event('__llgoLanguageExtensionReady'));`,
              () => resolve());
        }))
        .catch(error => {
          console.error('LLGo language extension registration failed', error);
          throw error;
        });
  }
})();
