# Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""GDB presentation adapter for the versioned LLGo debugger ABI."""

import json
from pathlib import Path
import re
import sys

import gdb


LLGO_DEBUGGER_MARKER_PREFIX = "__llgo_debugger_marker_v"
LLGO_DEBUGGER_SCHEMA_FILENAME = "llgo_debugger_schema_v1.json"
LLGO_MAX_STRING_SUMMARY_BYTES = 256
LLGO_MAX_TYPE_NAME_BYTES = 4096
LLGO_MAX_CONTAINER_SCAN_BUCKETS = 65536
LLGO_MAX_GOROUTINES = 65536


def _load_schema():
    source = Path(__file__).resolve()
    candidates = [source.with_name(LLGO_DEBUGGER_SCHEMA_FILENAME)]
    if len(source.parents) > 3:
        candidates.append(
            source.parents[3] / "internal" / "debugabi" / "schema_v1.json")
    errors = []
    for path in candidates:
        try:
            with path.open("r", encoding="utf-8") as schema_file:
                schema = json.load(schema_file)
            if schema.get("contract") != "llgo.debugger":
                raise ValueError("unexpected debugger schema contract")
            return schema, None
        except (OSError, ValueError, TypeError, json.JSONDecodeError) as error:
            errors.append(f"{path}: {error}")
    return {}, "; ".join(errors)


LLGO_DEBUGGER_SCHEMA, LLGO_DEBUGGER_SCHEMA_ERROR = _load_schema()
_RECORD_SCHEMA = LLGO_DEBUGGER_SCHEMA.get("record", {})
LLGO_DEBUGGER_RECORD_SYMBOL = _RECORD_SCHEMA.get("native_symbol", "")
LLGO_DEBUGGER_RECORD_SIZE = int(_RECORD_SCHEMA.get("size", 0))
try:
    LLGO_DEBUGGER_RECORD_MAGIC = bytes.fromhex(
        _RECORD_SCHEMA.get("magic_hex", ""))
except ValueError:
    LLGO_DEBUGGER_RECORD_MAGIC = b""
LLGO_DEBUGGER_RECORD_FIELDS = {
    field.get("name"): field
    for field in _RECORD_SCHEMA.get("fields", [])
    if isinstance(field, dict) and field.get("name")
}
LLGO_DEBUGGER_SCHEMAS = {
    symbol: (
        int(contract.get("schema_version", 0)),
        int(contract.get("runtime_layout_version", 0)),
        int(contract.get("llgo_abi_version", 0)),
    )
    for symbol, contract in _RECORD_SCHEMA.get("legacy_symbols", {}).items()
    if isinstance(contract, dict)
}
LLGO_BYTE_ORDERS = {
    int(value): name
    for value, name in LLGO_DEBUGGER_SCHEMA.get("byte_orders", {}).items()
}
LLGO_RUNTIME_LAYOUTS = {
    int(version): layout
    for version, layout in LLGO_DEBUGGER_SCHEMA.get(
        "runtime_layouts", {}).items()
    if isinstance(layout, dict)
}
_TARGET_INFO_CACHE = {}
_GOROUTINES_CACHE = {}


class TargetInfo:
    def __init__(self, marker_versions=(), schema_version=None,
                 runtime_layout_version=None, architecture="",
                 pointer_size=0, byte_order="unknown", record_version=None,
                 llgo_abi_version=None,
                 compatibility_error=None):
        self.marker_versions = marker_versions
        self.schema_version = schema_version
        self.runtime_layout_version = runtime_layout_version
        self.architecture = architecture
        self.pointer_size = pointer_size
        self.byte_order = byte_order
        self.record_version = record_version
        self.llgo_abi_version = llgo_abi_version
        self.compatibility_error = compatibility_error

    @property
    def supported(self):
        return (
            self.schema_version is not None and
            self.runtime_layout_version in LLGO_RUNTIME_LAYOUTS and
            self.compatibility_error is None
        )


def _symbol_address(name):
    try:
        output = gdb.execute(f"info address {name}", to_string=True)
    except gdb.error:
        return None
    match = re.search(r"\bat (?:address )?(0x[0-9a-fA-F]+)\b", output)
    return int(match.group(1), 16) if match else None


def _marker_versions(probe_known=True):
    versions = set()
    try:
        output = gdb.execute(
            f"info variables {LLGO_DEBUGGER_MARKER_PREFIX}", to_string=True)
    except gdb.error:
        output = ""
    pattern = re.compile(
        rf"\b{re.escape(LLGO_DEBUGGER_MARKER_PREFIX)}([0-9]+)\b")
    versions.update(
        int(match.group(1)) for match in pattern.finditer(output))
    # Some targets omit local object symbols from info variables. Probe only
    # known legacy versions in that case. Structured records already identify
    # their schema; enumerated unknown markers still detect conflicts without
    # hundreds of speculative info-address requests.
    if probe_known:
        known_versions = {1}
        known_versions.update(schema[0] for schema in LLGO_DEBUGGER_SCHEMAS.values())
        for version in sorted(known_versions - versions):
            if _symbol_address(
                    f"{LLGO_DEBUGGER_MARKER_PREFIX}{version}") is not None:
                versions.add(version)
    return tuple(sorted(versions))


def _record_field(raw, name):
    field = LLGO_DEBUGGER_RECORD_FIELDS.get(name)
    if not isinstance(field, dict):
        return None
    try:
        offset = int(field["offset"])
        size = int(field["size"])
    except (KeyError, TypeError, ValueError):
        return None
    if offset < 0 or size <= 0 or offset + size > len(raw):
        return None
    return int.from_bytes(raw[offset:offset + size], "little")


def _read_debugger_record():
    if not LLGO_DEBUGGER_RECORD_SYMBOL or LLGO_DEBUGGER_RECORD_SIZE <= 0:
        return None
    address = _symbol_address(LLGO_DEBUGGER_RECORD_SYMBOL)
    if address is None:
        return None
    try:
        memory = gdb.selected_inferior().read_memory(
            address, LLGO_DEBUGGER_RECORD_SIZE)
        return memory.tobytes()
    except (gdb.error, ValueError):
        return None


def _decode_debugger_record(raw):
    if len(raw) != LLGO_DEBUGGER_RECORD_SIZE:
        return None, "incorrectly sized native record"
    if not LLGO_DEBUGGER_RECORD_MAGIC or not raw.startswith(
            LLGO_DEBUGGER_RECORD_MAGIC):
        return None, "invalid record magic"
    values = {
        name: _record_field(raw, name)
        for name in (
            "record_version", "schema_version", "runtime_layout_version",
            "llgo_abi_version", "reserved_abi", "pointer_size", "byte_order",
            "reserved",
        )
    }
    if any(value is None for value in values.values()):
        return None, "record fields do not match the loaded schema"
    if values["reserved"] != 0 or values["reserved_abi"] != 0:
        return None, "record reserved byte is non-zero"
    return values, None


def _target_byte_order():
    try:
        output = gdb.execute("show endian", to_string=True).lower()
    except gdb.error:
        return "unknown"
    if "little endian" in output:
        return "little"
    if "big endian" in output:
        return "big"
    return "unknown"


def _target_cache_key():
    try:
        objfiles = tuple(objfile.filename or "" for objfile in gdb.objfiles())
        inferior = gdb.selected_inferior()
        architecture = inferior.architecture().name()
        pid = inferior.pid
    except (gdb.error, AttributeError):
        return ()
    return objfiles, architecture, inferior.num, pid


def inspect_target():
    key = _target_cache_key()
    cached = _TARGET_INFO_CACHE.get(key)
    if cached is not None:
        return cached

    raw_record = _read_debugger_record()
    marker_versions = _marker_versions(probe_known=raw_record is None)
    schema_version = None
    runtime_layout_version = None
    llgo_abi_version = None
    record_version = None
    compatibility_error = None
    try:
        pointer_size = gdb.lookup_type("void").pointer().sizeof
        architecture = gdb.selected_inferior().architecture().name()
    except (gdb.error, AttributeError):
        pointer_size = 0
        architecture = ""
    byte_order = _target_byte_order()

    if raw_record is not None:
        record, compatibility_error = _decode_debugger_record(raw_record)
        if record is not None:
            record_version = record["record_version"]
            schema_version = record["schema_version"]
            runtime_layout_version = record["runtime_layout_version"]
            llgo_abi_version = record["llgo_abi_version"]
            record_byte_order = LLGO_BYTE_ORDERS.get(record["byte_order"])
            expected = (
                int(_RECORD_SCHEMA.get("version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get("schema_version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get(
                    "runtime_layout_version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get("llgo_abi_version", 0)),
            )
            actual = (
                record_version, schema_version, runtime_layout_version,
                llgo_abi_version,
            )
            if actual != expected:
                compatibility_error = (
                    "unsupported record/schema/runtime/ABI versions "
                    f"{actual}, want {expected}")
            elif record["pointer_size"] != pointer_size:
                compatibility_error = (
                    f"record pointer size {record['pointer_size']} does not "
                    f"match target pointer size {pointer_size}")
            elif record_byte_order != byte_order:
                compatibility_error = (
                    f"record byte order "
                    f"{record_byte_order or record['byte_order']} does not "
                    f"match target byte order {byte_order}")
            elif marker_versions and marker_versions != (schema_version,):
                compatibility_error = (
                    f"record schema v{schema_version} conflicts with legacy "
                    f"marker version(s) {marker_versions}")

    if raw_record is None and len(marker_versions) == 1:
        candidate = marker_versions[0]
        for supported in LLGO_DEBUGGER_SCHEMAS.values():
            if candidate == supported[0]:
                schema_version = supported[0]
                runtime_layout_version = supported[1]
                llgo_abi_version = supported[2]
                break

    if ((marker_versions or raw_record is not None) and
            not LLGO_DEBUGGER_SCHEMA and compatibility_error is None):
        compatibility_error = (
            "debugger schema could not be loaded: " +
            (LLGO_DEBUGGER_SCHEMA_ERROR or "unknown error"))

    info = TargetInfo(
        marker_versions=marker_versions,
        schema_version=schema_version,
        runtime_layout_version=runtime_layout_version,
        architecture=architecture,
        pointer_size=pointer_size,
        byte_order=byte_order,
        record_version=record_version,
        llgo_abi_version=llgo_abi_version,
        compatibility_error=compatibility_error,
    )
    _TARGET_INFO_CACHE[key] = info
    return info


def target_status(info):
    if not info.marker_versions and info.record_version is None:
        return "Not an LLGo target; raw GDB debugging remains available."
    if info.compatibility_error:
        return (
            f"Unsupported LLGo debugger ABI: {info.compatibility_error}; "
            "raw GDB debugging remains available."
        )
    if not info.supported:
        versions = ", ".join(f"v{version}"
                             for version in info.marker_versions)
        return (
            f"Unsupported LLGo debugger marker version(s): {versions}; "
            "raw GDB debugging remains available."
        )
    abi = (f"LLGo ABI v{info.llgo_abi_version}; "
           if info.llgo_abi_version is not None else "")
    return (
        f"LLGo debugger schema v{info.schema_version} "
        f"(runtime layout v{info.runtime_layout_version}); "
        f"{abi}target {info.architecture}; "
        f"pointer size {info.pointer_size}; byte order {info.byte_order}."
    )


def _require_supported_target():
    info = inspect_target()
    if not info.supported:
        raise gdb.GdbError(target_status(info))
    return info


def _layout_for_value(_value=None):
    info = inspect_target()
    return (LLGO_RUNTIME_LAYOUTS.get(info.runtime_layout_version)
            if info.supported else None)


def _raw_value(value):
    try:
        while value.type.code in (gdb.TYPE_CODE_REF, gdb.TYPE_CODE_RVALUE_REF):
            value = value.referenced_value()
    except (gdb.error, AttributeError):
        pass
    return value


def _canonical_type_name(value):
    try:
        value_type = value.type.strip_typedefs()
    except (gdb.error, AttributeError):
        value_type = value.type
    name = str(value_type)
    if name.startswith("struct "):
        return name[len("struct "):]
    return name


def _pointer_target_name(value):
    try:
        value_type = value.type.strip_typedefs()
        if value_type.code != gdb.TYPE_CODE_PTR:
            return ""
        name = str(value_type.target().strip_typedefs())
    except (gdb.error, AttributeError):
        return ""
    return name[len("struct "):] if name.startswith("struct ") else name


def _field(value, name):
    try:
        return value[name]
    except (gdb.error, KeyError, TypeError):
        return None


def _value_as_int(value):
    if value is None:
        return None
    try:
        return int(value)
    except (gdb.error, ValueError, TypeError):
        return None


def _lookup_type(name):
    for candidate in (name, "struct " + name):
        try:
            return gdb.lookup_type(candidate)
        except gdb.error:
            pass
    return None


def _string_fields(value, layout):
    raw = _raw_value(value)
    if _canonical_type_name(raw) != layout["string"]["type_name"]:
        return None
    data = _field(raw, layout["string"]["data"])
    length = _value_as_int(_field(raw, layout["string"]["length"]))
    address = _value_as_int(data)
    if (data is None or length is None or length < 0 or
            length > LLGO_MAX_TYPE_NAME_BYTES or
            (length != 0 and (address is None or address == 0))):
        return None
    return data, length


def _read_string(value, layout, limit=LLGO_MAX_STRING_SUMMARY_BYTES):
    fields = _string_fields(value, layout)
    if fields is None:
        return None
    data, length = fields
    truncated = length > limit
    length = min(length, limit)
    try:
        text = data.string(encoding="utf-8", errors="replace", length=length)
    except (gdb.error, UnicodeError):
        return None
    return text + ("..." if truncated else "")


class StringPrinter:
    def __init__(self, value, layout):
        self.value = value
        self.layout = layout

    def display_hint(self):
        return "string"

    def to_string(self):
        return _read_string(self.value, self.layout)


class SlicePrinter:
    def __init__(self, value, layout):
        self.value = _raw_value(value)
        self.layout = layout

    def _fields(self):
        spec = self.layout["slice"]
        if re.fullmatch(spec["type_pattern"],
                        _canonical_type_name(self.value)) is None:
            return None
        data = _field(self.value, spec["data"])
        length = _value_as_int(_field(self.value, spec["length"]))
        capacity = _value_as_int(_field(self.value, spec["capacity"]))
        address = _value_as_int(data)
        if (data is None or length is None or capacity is None or
                length < 0 or capacity < length or
                (length and (address is None or address == 0))):
            return None
        try:
            element_type = data.type.strip_typedefs().target()
        except (gdb.error, AttributeError):
            return None
        return data.cast(element_type.pointer()), length, capacity

    def display_hint(self):
        return "array"

    def to_string(self):
        fields = self._fields()
        if fields is None:
            return None
        return f"len={fields[1]} cap={fields[2]}"

    def children(self):
        fields = self._fields()
        if fields is None:
            return
        data, length, _ = fields
        for index in range(length):
            try:
                yield f"[{index}]", (data + index).dereference()
            except gdb.error:
                return


def _runtime_type_name(value, address, layout):
    runtime = layout["runtime_type"]
    runtime_type = _lookup_type(runtime["type_name"])
    if runtime_type is None:
        return None
    try:
        runtime_value = gdb.Value(address).cast(
            runtime_type.pointer()).dereference()
    except gdb.error:
        return None
    name = _read_string(_field(runtime_value, runtime["string"]),
                        layout, LLGO_MAX_TYPE_NAME_BYTES)
    if name is None:
        return None
    tflag = _value_as_int(_field(runtime_value, runtime["tflag"]))
    if tflag is not None and tflag & int(runtime["extra_star_flag"]):
        name = "*" + name
    return name


class InterfacePrinter:
    def __init__(self, value, layout):
        self.value = _raw_value(value)
        self.layout = layout

    def to_string(self):
        interface = self.layout["interface"]
        name = _canonical_type_name(self.value)
        if re.fullmatch(interface["type_pattern"], name) is None:
            return None
        type_address = _value_as_int(_field(self.value, interface["type"]))
        if type_address is None:
            return None
        if type_address == 0:
            return "nil"
        if name != interface["empty_type"]:
            itab_type = _lookup_type(interface["itab_type"])
            if itab_type is None:
                return None
            try:
                itab = gdb.Value(type_address).cast(
                    itab_type.pointer()).dereference()
            except gdb.error:
                return None
            type_address = _value_as_int(
                _field(itab, interface["itab_concrete_type"]))
            if type_address is None or type_address == 0:
                return None
        type_name = _runtime_type_name(
            self.value, type_address, self.layout)
        return f"type={type_name}" if type_name else None


def _symbol_name(address):
    if not address:
        return None
    try:
        block = gdb.block_for_pc(address)
        if block is not None and block.function is not None:
            return block.function.print_name
    except (gdb.error, RuntimeError):
        pass
    try:
        output = gdb.execute(f"info symbol 0x{address:x}", to_string=True)
    except gdb.error:
        return None
    if output.startswith("No symbol"):
        return None
    return output.split(" in section ", 1)[0].split(" + ", 1)[0].strip()


class FunctionPrinter:
    def __init__(self, value, layout):
        self.value = _raw_value(value)
        self.layout = layout

    def to_string(self):
        function = self.layout["function"]
        raw = self.value
        code_field, data_field = function["code"], function["data"]
        descriptor = function.get("descriptor_type_pattern", "(?!)")
        if re.fullmatch(descriptor, _pointer_target_name(raw)):
            address = _value_as_int(raw)
            if address is None:
                return None
            if address == 0:
                return "nil"
            raw = raw.dereference()
            code_field, data_field = function["descriptor_code"], function["descriptor_data"]
        elif re.fullmatch(function["type_pattern"], _canonical_type_name(raw)) is None:
            return None
        code = _value_as_int(_field(raw, code_field))
        data = _value_as_int(_field(raw, data_field))
        if code is None or data is None:
            return None
        if code == 0:
            return "nil"
        name = _symbol_name(code) or f"0x{code:x}"
        if name.startswith("__llgo_stub."):
            name = name[len("__llgo_stub."):]
        if name.endswith(function["bound_symbol_suffix"]):
            return f"{name} (bound method)"
        if re.search(function["closure_symbol_pattern"], name):
            return f"{name} (closure)"
        return name


def _pointer_runtime_value(value, pattern, target_prefix=""):
    raw = _raw_value(value)
    matches_name = (
        re.fullmatch(pattern, _canonical_type_name(raw)) is not None)
    matches_target = (
        target_prefix and
        _pointer_target_name(raw).startswith(target_prefix)
    )
    if not matches_name and not matches_target:
        return None, None
    address = _value_as_int(raw)
    if address is None or address == 0:
        return address, None
    try:
        return address, raw.dereference()
    except gdb.error:
        return address, None


def _array_length(value):
    try:
        low, high = value.type.strip_typedefs().range()
        return high - low + 1
    except (gdb.error, AttributeError, TypeError):
        return 0


class MapPrinter:
    def __init__(self, value, layout):
        self.value = _raw_value(value)
        self.layout = layout

    def _hash(self):
        return _pointer_runtime_value(
            self.value, self.layout["map"]["type_pattern"], "hash<")

    def display_hint(self):
        return "map"

    def to_string(self):
        address, hash_value = self._hash()
        if address is None:
            return None
        if address == 0:
            return "nil"
        if hash_value is None:
            return None
        length = _value_as_int(
            _field(hash_value, self.layout["map"]["count"]))
        return f"len={length}" if length is not None and length >= 0 else None

    def children(self):
        _, hash_value = self._hash()
        if hash_value is None:
            return
        spec = self.layout["map"]
        length = _value_as_int(_field(hash_value, spec["count"]))
        flags = _value_as_int(_field(hash_value, spec["flags"]))
        bucket_bits = _value_as_int(
            _field(hash_value, spec["bucket_bits"]))
        buckets = _field(hash_value, spec["buckets"])
        old_buckets = _field(hash_value, spec["old_buckets"])
        buckets_address = _value_as_int(buckets)
        old_buckets_address = _value_as_int(old_buckets)
        if (length is None or flags is None or bucket_bits is None or
                buckets is None or old_buckets is None or
                buckets_address is None or old_buckets_address is None or
                length <= 0 or bucket_bits < 0 or bucket_bits >= 63 or
                buckets_address == 0):
            return
        try:
            bucket_type = buckets.type.strip_typedefs().target()
            bucket_size = bucket_type.sizeof
        except (gdb.error, AttributeError):
            return
        if bucket_size <= 0:
            return

        logical_buckets = 1 << bucket_bits
        scan_buckets = min(logical_buckets,
                           LLGO_MAX_CONTAINER_SCAN_BUCKETS)
        old_count = (
            logical_buckets
            if flags & int(spec["same_size_grow_flag"])
            else logical_buckets >> 1
        )
        emitted = 0

        def bucket_at(address):
            try:
                return gdb.Value(address).cast(
                    bucket_type.pointer()).dereference()
            except gdb.error:
                return None

        def evacuated(address):
            bucket = bucket_at(address)
            if bucket is None:
                return False
            tophash = _field(bucket, spec["bucket_tophash"])
            if tophash is None or _array_length(tophash) == 0:
                return False
            first = _value_as_int(tophash[0])
            return (
                first is not None and
                int(spec["evacuated_tophash_min"]) <= first <=
                int(spec["evacuated_tophash_max"])
            )

        for bucket_index in range(scan_buckets):
            bucket_address = buckets_address + bucket_index * bucket_size
            if old_buckets_address and old_count:
                old_index = bucket_index & (old_count - 1)
                old_address = (
                    old_buckets_address + old_index * bucket_size)
                if not evacuated(old_address):
                    if bucket_index >= old_count:
                        continue
                    bucket_address = old_address

            visited = set()
            while (bucket_address and bucket_address not in visited and
                   emitted < length):
                visited.add(bucket_address)
                bucket = bucket_at(bucket_address)
                if bucket is None:
                    return
                tophash = _field(bucket, spec["bucket_tophash"])
                keys = _field(bucket, spec["bucket_keys"])
                indirect_keys = keys is None
                if indirect_keys:
                    keys = _field(bucket, spec["bucket_indirect_keys"])
                values = _field(bucket, spec["bucket_values"])
                indirect_values = values is None
                if indirect_values:
                    values = _field(
                        bucket, spec["bucket_indirect_values"])
                if tophash is None or keys is None or values is None:
                    return
                slots = min(
                    _array_length(tophash),
                    _array_length(keys),
                    _array_length(values),
                )
                for slot in range(slots):
                    top = _value_as_int(tophash[slot])
                    if (top is None or
                            top < int(spec["occupied_tophash_min"])):
                        continue
                    try:
                        key = keys[slot]
                        element = values[slot]
                        if indirect_keys and (_value_as_int(key) or 0) != 0:
                            key = key.dereference()
                        if (indirect_values and
                                (_value_as_int(element) or 0) != 0):
                            element = element.dereference()
                    except gdb.error:
                        return
                    yield f"key[{emitted}]", key
                    yield f"value[{emitted}]", element
                    emitted += 1
                    if emitted >= length:
                        return
                bucket_address = (
                    _value_as_int(_field(bucket, spec["bucket_overflow"]))
                    or 0
                )


class ChannelPrinter:
    def __init__(self, value, layout):
        self.value = _raw_value(value)
        self.layout = layout

    def _fields(self):
        spec = self.layout["channel"]
        address, channel = _pointer_runtime_value(
            self.value, spec["type_pattern"], "hchan<")
        if address is None or address == 0 or channel is None:
            return address, None
        length = _value_as_int(_field(channel, spec["count"]))
        capacity = _value_as_int(_field(channel, spec["capacity"]))
        buffer = _value_as_int(_field(channel, spec["buffer"]))
        receive_index = _value_as_int(
            _field(channel, spec["receive_index"]))
        closed = _value_as_int(_field(channel, spec["closed"]))
        if (length is None or capacity is None or buffer is None or
                receive_index is None or closed is None or length < 0 or
                capacity < length or
                (capacity and receive_index >= capacity) or
                (length and buffer == 0)):
            return address, None
        try:
            queue = _field(channel, spec["receive_queue"])
            first = _field(queue, spec["queue_first"])
            waiter_type = first.type.strip_typedefs().target()
            element_pointer = next(
                field.type
                for field in waiter_type.strip_typedefs().fields()
                if field.name == spec["waiter_element"]
            )
            element_type = element_pointer.strip_typedefs().target()
        except (gdb.error, AttributeError, StopIteration):
            return address, None
        return address, (
            length, capacity, buffer, receive_index, closed != 0,
            element_type,
        )

    def display_hint(self):
        return "array"

    def to_string(self):
        address, fields = self._fields()
        if address is None:
            return None
        if address == 0:
            return "nil"
        if fields is None:
            return None
        suffix = " closed" if fields[4] else ""
        return f"len={fields[0]} cap={fields[1]}{suffix}"

    def children(self):
        _, fields = self._fields()
        if fields is None:
            return
        length, capacity, buffer, receive_index, _, element_type = fields
        if capacity == 0 or buffer == 0:
            return
        pointer = gdb.Value(buffer).cast(element_type.pointer())
        for index in range(length):
            buffer_index = (receive_index + index) % capacity
            try:
                yield f"[{index}]", (
                    pointer + buffer_index).dereference()
            except gdb.error:
                return


def lookup_pretty_printer(value):
    layout = _layout_for_value(value)
    if layout is None:
        return None
    raw = _raw_value(value)
    name = _canonical_type_name(raw)
    try:
        if name == layout["string"]["type_name"]:
            return StringPrinter(raw, layout)
        if re.fullmatch(layout["slice"]["type_pattern"], name):
            return SlicePrinter(raw, layout)
        if re.fullmatch(layout["interface"]["type_pattern"], name):
            return InterfacePrinter(raw, layout)
        if (re.fullmatch(layout["function"]["type_pattern"], name) or
                re.fullmatch(layout["function"].get("descriptor_type_pattern", "(?!)"),
                             _pointer_target_name(raw))):
            return FunctionPrinter(raw, layout)
        if (re.fullmatch(layout["map"]["type_pattern"], name) or
                _pointer_target_name(raw).startswith("hash<")):
            return MapPrinter(raw, layout)
        if (re.fullmatch(layout["channel"]["type_pattern"], name) or
                _pointer_target_name(raw).startswith("hchan<")):
            return ChannelPrinter(raw, layout)
    except (gdb.error, KeyError, TypeError, ValueError):
        return None
    return None


lookup_pretty_printer.name = "llgo"


def _goroutine_layout(info):
    layout = LLGO_RUNTIME_LAYOUTS.get(info.runtime_layout_version)
    return layout.get("goroutine") if layout else None


def _read_pointer(address, pointer_size, byte_order):
    if byte_order not in ("little", "big") or pointer_size not in (4, 8):
        return None
    try:
        raw = gdb.selected_inferior().read_memory(
            address, pointer_size).tobytes()
        return int.from_bytes(raw, byte_order)
    except (gdb.error, ValueError):
        return None


def _darwin_thread_id(port):
    # Darwin GDB's native ptid contains a Mach port in the debugger's task,
    # while the runtime stores pthread_threadid_np's 64-bit thread_id.
    # Query the kernel from this debugger process; never call the inferior.
    try:
        import ctypes

        class ThreadIdentifierInfo(ctypes.Structure):
            _fields_ = [(name, ctypes.c_uint64) for name in (
                "thread_id", "thread_handle", "dispatch_qaddr")]

        system = ctypes.CDLL("/usr/lib/libSystem.B.dylib")
        thread_info = system.thread_info
        thread_info.argtypes = [ctypes.c_uint32, ctypes.c_int,
                                ctypes.c_void_p, ctypes.POINTER(ctypes.c_uint32)]
        thread_info.restype = ctypes.c_int
        info = ThreadIdentifierInfo()
        count = ctypes.c_uint32(ctypes.sizeof(info) // ctypes.sizeof(ctypes.c_uint32))
        if thread_info(port, 4, ctypes.byref(info), ctypes.byref(count)) != 0:
            return None
        if count.value < 6:
            return None
        return info.thread_id or None
    except (AttributeError, ImportError, OSError, TypeError, ValueError):
        return None


def _threads_by_procid():
    try:
        inferior = gdb.selected_inferior()
        threads = inferior.threads()
        connection = getattr(inferior, "connection", None)
        local_darwin = (sys.platform == "darwin" and
                        getattr(connection, "type", None) == "native")
    except gdb.error:
        return {}
    result = {}
    for thread in threads:
        if local_darwin:
            # Do not mistake the local Mach port for a system thread ID if
            # thread_info fails, or reinterpret remote target thread IDs.
            procid = _darwin_thread_id(thread.ptid[2])
            if procid:
                result[procid] = thread
        else:
            for procid in thread.ptid[1:]:
                if procid > 0:
                    result[procid] = thread
    return result


def _thread_for_procid(procid, threads=None):
    if procid <= 0:
        return None
    if threads is None:
        threads = _threads_by_procid()
    return threads.get(procid)


def _invalidate_caches(_event=None):
    _TARGET_INFO_CACHE.clear()
    _GOROUTINES_CACHE.clear()


def _goroutines():
    info = _require_supported_target()
    key = _target_cache_key()
    cached = _GOROUTINES_CACHE.get(key)
    if cached is not None:
        return cached
    layout = _goroutine_layout(info)
    if not layout:
        raise gdb.GdbError(
            "LLGo debugger schema has no goroutine layout.")
    head_address = _symbol_address(layout["head_symbol"])
    goroutine_type = _lookup_type(layout["goroutine_type"])
    if head_address is None or goroutine_type is None:
        raise gdb.GdbError(
            "LLGo goroutine registry is unavailable in this target.")
    pointer = _read_pointer(
        head_address, info.pointer_size, info.byte_order)
    if pointer is None:
        raise gdb.GdbError(
            "LLGo goroutine registry requires a stopped process.")

    result = []
    visited = set()
    status_names = {
        int(status): name
        for status, name in layout.get("status_names", {}).items()
    }
    threads = _threads_by_procid()
    while pointer and pointer not in visited:
        if len(result) >= LLGO_MAX_GOROUTINES:
            raise gdb.GdbError(
                f"LLGo goroutine registry exceeds "
                f"{LLGO_MAX_GOROUTINES} entries.")
        visited.add(pointer)
        try:
            value = gdb.Value(pointer).cast(
                goroutine_type.pointer()).dereference()
        except gdb.error as error:
            raise gdb.GdbError(
                f"cannot read LLGo goroutine at 0x{pointer:x}: {error}")
        status = _value_as_int(_field(value, layout["status"]))
        goid = _value_as_int(_field(value, layout["id"]))
        parent = _value_as_int(_field(value, layout["parent_id"]))
        procid = _value_as_int(_field(value, layout["procid"]))
        result.append({
            "address": pointer,
            "goid": goid if goid is not None else -1,
            "parent": parent if parent is not None else -1,
            "status": status if status is not None else -1,
            "status_name": status_names.get(
                status, f"unknown({status})"),
            "procid": procid if procid is not None else -1,
            "thread": _thread_for_procid(
                procid if procid is not None else -1, threads),
        })
        pointer = _value_as_int(_field(value, layout["next"])) or 0
    _GOROUTINES_CACHE[key] = result
    return result


class LLGoPrefix(gdb.Command):
    """LLGo debugger commands."""

    def __init__(self):
        super().__init__("llgo", gdb.COMMAND_USER, prefix=True)


class LLGoStatus(gdb.Command):
    """Show LLGo debugger ABI and target compatibility."""

    def __init__(self):
        super().__init__("llgo status", gdb.COMMAND_STATUS)

    def invoke(self, _argument, _from_tty):
        print(target_status(inspect_target()))


class LLGoPrint(gdb.Command):
    """Print an expression with LLGo runtime presentation."""

    def __init__(self):
        super().__init__("llgo print", gdb.COMMAND_DATA,
                         gdb.COMPLETE_EXPRESSION)

    def invoke(self, argument, _from_tty):
        _require_supported_target()
        expression = argument.strip()
        if not expression:
            raise gdb.GdbError("usage: llgo print <expression>")
        gdb.execute("print " + expression)


class LLGoVars(gdb.Command):
    """Print arguments and locals with LLGo runtime presentation."""

    def __init__(self):
        super().__init__("llgo vars", gdb.COMMAND_DATA)

    def invoke(self, _argument, _from_tty):
        _require_supported_target()
        if gdb.selected_thread() is None:
            raise gdb.GdbError(
                "LLGo command requires a stopped process.")
        gdb.execute("info args")
        gdb.execute("info locals")


class LLGoGoroutines(gdb.Command):
    """List LLGo goroutines and their native threads."""

    def __init__(self):
        super().__init__("llgo goroutines", gdb.COMMAND_STACK)

    def invoke(self, _argument, _from_tty):
        for goroutine in _goroutines():
            thread = goroutine["thread"]
            thread_id = str(thread.num) if thread is not None else "unavailable"
            print(
                f"goroutine {goroutine['goid']} "
                f"[{goroutine['status_name']}] "
                f"parent={goroutine['parent']} "
                f"tid={goroutine['procid']} thread={thread_id}"
            )


class LLGoGoroutine(gdb.Command):
    """Run a GDB command in the native thread that owns an LLGo goroutine."""

    def __init__(self):
        super().__init__("llgo goroutine", gdb.COMMAND_STACK)

    def invoke(self, argument, _from_tty):
        args = gdb.string_to_argv(argument)
        if len(args) < 2:
            raise gdb.GdbError(
                "usage: llgo goroutine <id> <gdb command>")
        try:
            goid = int(args[0], 10)
        except ValueError as error:
            raise gdb.GdbError(
                f"invalid goroutine id {args[0]!r}") from error
        goroutine = next(
            (item for item in _goroutines() if item["goid"] == goid),
            None,
        )
        if goroutine is None:
            raise gdb.GdbError(f"LLGo goroutine {goid} was not found.")
        thread = goroutine["thread"]
        if thread is None:
            raise gdb.GdbError(
                f"LLGo goroutine {goid} has no matching debugger thread.")
        current = gdb.selected_thread()
        try:
            thread.switch()
            gdb.execute(" ".join(args[1:]))
        finally:
            if current is not None and current.is_valid():
                current.switch()


def register():
    # Reuse a stopped-process snapshot for goroutine backtraces. Resume/stop,
    # debugger writes and thread/objfile changes all invalidate target data.
    for name in (
        "stop", "cont", "exited", "new_objfile", "clear_objfiles",
        "inferior_deleted", "memory_changed", "register_changed",
        "new_thread", "thread_exited",
    ):
        event = getattr(gdb.events, name, None)
        if event is not None:
            event.connect(_invalidate_caches)
    gdb.pretty_printers[:] = [
        printer for printer in gdb.pretty_printers
        if getattr(printer, "name", None) != "llgo"
    ]
    gdb.pretty_printers.append(lookup_pretty_printer)
    LLGoPrefix()
    LLGoStatus()
    LLGoPrint()
    LLGoVars()
    LLGoGoroutines()
    LLGoGoroutine()


register()
