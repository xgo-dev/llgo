# pylint: disable=missing-module-docstring,missing-class-docstring,missing-function-docstring

from dataclasses import dataclass
import ctypes
import json
from pathlib import Path
from typing import List, Optional, Dict, Any, Tuple
import re
import sys
import lldb


LLGO_DEBUGGER_MARKER_PREFIX = "__llgo_debugger_marker_v"
LLGO_DEBUGGER_SCHEMA_FILENAME = "llgo_debugger_schema_v1.json"
LLGO_TYPE_CATEGORY = "LLGo"
LLGO_MAX_STRING_SUMMARY_BYTES = 256
LLGO_MAX_TYPE_NAME_BYTES = 4096
LLGO_DEFAULT_MAX_CHILDREN = 256
LLGO_MAX_CONTAINER_SCAN_BUCKETS = 65536
LLGO_MAX_GOROUTINES = 65536
LLGO_MAX_STACK_FRAMES = 256
_TARGET_INFO_CACHE: Dict[Tuple[Any, ...], "LLGoTargetInfo"] = {}
_CODE_ADDRESS_HOOK_TARGETS = set()


def _load_debugger_schema() -> Tuple[Dict[str, Any], Optional[str]]:
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


LLGO_DEBUGGER_SCHEMA, LLGO_DEBUGGER_SCHEMA_ERROR = _load_debugger_schema()
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


@dataclass(frozen=True)
class LLGoRuntimeLayout:
    string_type: str
    string_data: str
    string_len: str
    slice_type_pattern: str
    slice_data: str
    slice_len: str
    slice_cap: str
    interface_type_pattern: str
    interface_type: str
    interface_data: str
    empty_interface_type: str
    runtime_itab_type: str
    runtime_itab_concrete_type: str
    runtime_type: str
    runtime_type_tflag: str
    runtime_type_extra_star_flag: int
    runtime_type_string: str
    function_type_pattern: str
    function_code: str
    function_data: str
    function_descriptor_type_pattern: str
    function_descriptor_code: str
    function_descriptor_data: str
    function_closure_symbol_pattern: str
    function_bound_symbol_suffix: str
    map_type_pattern: str
    map_count: str
    map_flags: str
    map_same_size_grow_flag: int
    map_bucket_bits: str
    map_buckets: str
    map_old_buckets: str
    map_bucket_tophash: str
    map_bucket_keys: str
    map_bucket_indirect_keys: str
    map_bucket_values: str
    map_bucket_indirect_values: str
    map_bucket_overflow: str
    map_evacuated_tophash_min: int
    map_evacuated_tophash_max: int
    map_occupied_tophash_min: int
    channel_type_pattern: str
    channel_count: str
    channel_capacity: str
    channel_buffer: str
    channel_closed: str
    channel_receive_index: str
    channel_receive_queue: str
    channel_queue_first: str
    channel_waiter_element: str


@dataclass(frozen=True)
class LLGoSliceValue:
    address: int
    length: int
    capacity: int
    element_type: lldb.SBType
    element_size: int


@dataclass(frozen=True)
class LLGoChannelValue:
    length: int
    capacity: int
    buffer: int
    receive_index: int
    closed: bool
    element_type: lldb.SBType
    element_size: int


@dataclass(frozen=True)
class LLGoGoroutineLayout:
    head_symbol: str
    goroutine_type: str
    goroutine_next: str
    goroutine_status: str
    goroutine_id: str
    goroutine_parent_id: str
    procid: str
    status_names: Tuple[Tuple[int, str], ...]


@dataclass(frozen=True)
class LLGoGoroutineValue:
    address: int
    goid: int
    parent_goid: int
    status: int
    status_name: str
    procid: int


def _runtime_layouts() -> Dict[int, LLGoRuntimeLayout]:
    layouts = {}
    for version, raw in LLGO_DEBUGGER_SCHEMA.get(
            "runtime_layouts", {}).items():
        string_layout = raw.get("string", {})
        slice_layout = raw.get("slice", {})
        interface_layout = raw.get("interface", {})
        runtime_type_layout = raw.get("runtime_type", {})
        function_layout = raw.get("function", {})
        map_layout = raw.get("map", {})
        channel_layout = raw.get("channel", {})
        try:
            layouts[int(version)] = LLGoRuntimeLayout(
                string_type=string_layout["type_name"],
                string_data=string_layout["data"],
                string_len=string_layout["length"],
                slice_type_pattern=slice_layout["type_pattern"],
                slice_data=slice_layout["data"],
                slice_len=slice_layout["length"],
                slice_cap=slice_layout["capacity"],
                interface_type_pattern=interface_layout["type_pattern"],
                interface_type=interface_layout["type"],
                interface_data=interface_layout["data"],
                empty_interface_type=interface_layout["empty_type"],
                runtime_itab_type=interface_layout["itab_type"],
                runtime_itab_concrete_type=(
                    interface_layout["itab_concrete_type"]),
                runtime_type=runtime_type_layout["type_name"],
                runtime_type_tflag=runtime_type_layout["tflag"],
                runtime_type_extra_star_flag=(
                    runtime_type_layout["extra_star_flag"]),
                runtime_type_string=runtime_type_layout["string"],
                function_type_pattern=function_layout["type_pattern"],
                function_code=function_layout["code"],
                function_data=function_layout["data"],
                function_descriptor_type_pattern=function_layout.get("descriptor_type_pattern", "(?!)"),
                function_descriptor_code=function_layout.get("descriptor_code", ""),
                function_descriptor_data=function_layout.get("descriptor_data", ""),
                function_closure_symbol_pattern=(
                    function_layout["closure_symbol_pattern"]),
                function_bound_symbol_suffix=(
                    function_layout["bound_symbol_suffix"]),
                map_type_pattern=map_layout["type_pattern"],
                map_count=map_layout["count"],
                map_flags=map_layout["flags"],
                map_same_size_grow_flag=map_layout[
                    "same_size_grow_flag"],
                map_bucket_bits=map_layout["bucket_bits"],
                map_buckets=map_layout["buckets"],
                map_old_buckets=map_layout["old_buckets"],
                map_bucket_tophash=map_layout["bucket_tophash"],
                map_bucket_keys=map_layout["bucket_keys"],
                map_bucket_indirect_keys=(
                    map_layout["bucket_indirect_keys"]),
                map_bucket_values=map_layout["bucket_values"],
                map_bucket_indirect_values=(
                    map_layout["bucket_indirect_values"]),
                map_bucket_overflow=map_layout["bucket_overflow"],
                map_evacuated_tophash_min=map_layout[
                    "evacuated_tophash_min"],
                map_evacuated_tophash_max=map_layout[
                    "evacuated_tophash_max"],
                map_occupied_tophash_min=map_layout[
                    "occupied_tophash_min"],
                channel_type_pattern=channel_layout["type_pattern"],
                channel_count=channel_layout["count"],
                channel_capacity=channel_layout["capacity"],
                channel_buffer=channel_layout["buffer"],
                channel_closed=channel_layout["closed"],
                channel_receive_index=channel_layout["receive_index"],
                channel_receive_queue=channel_layout["receive_queue"],
                channel_queue_first=channel_layout["queue_first"],
                channel_waiter_element=channel_layout["waiter_element"],
            )
        except (KeyError, TypeError, ValueError):
            continue
    return layouts


LLGO_RUNTIME_LAYOUTS = _runtime_layouts()


def _goroutine_layouts() -> Dict[int, LLGoGoroutineLayout]:
    layouts = {}
    for version, raw in LLGO_DEBUGGER_SCHEMA.get(
            "runtime_layouts", {}).items():
        goroutine = raw.get("goroutine", {})
        try:
            status_names = tuple(sorted(
                (int(status), str(name))
                for status, name in goroutine["status_names"].items()
            ))
            layouts[int(version)] = LLGoGoroutineLayout(
                head_symbol=goroutine["head_symbol"],
                goroutine_type=goroutine["goroutine_type"],
                goroutine_next=goroutine["next"],
                goroutine_status=goroutine["status"],
                goroutine_id=goroutine["id"],
                goroutine_parent_id=goroutine["parent_id"],
                procid=goroutine["procid"],
                status_names=status_names,
            )
        except (AttributeError, KeyError, TypeError, ValueError):
            continue
    return layouts


LLGO_GOROUTINE_LAYOUTS = _goroutine_layouts()


@dataclass(frozen=True)
class LLGoDebuggerRecord:
    record_version: int
    schema_version: int
    runtime_layout_version: int
    llgo_abi_version: int
    pointer_size: int
    byte_order: int


@dataclass(frozen=True)
class LLGoTargetInfo:
    marker_versions: Tuple[int, ...]
    schema_version: Optional[int]
    runtime_layout_version: Optional[int]
    triple: str
    pointer_size: int
    byte_order: str
    record_version: Optional[int] = None
    llgo_abi_version: Optional[int] = None
    compatibility_error: Optional[str] = None

    @property
    def supported(self) -> bool:
        return (self.schema_version is not None and
                self.compatibility_error is None)


def log(*args: Any, **kwargs: Any) -> None:
    print(*args, **kwargs, flush=True)


def __lldb_init_module(debugger: lldb.SBDebugger, _: Dict[str, Any]) -> None:
    register_commands(debugger)


def register_commands(debugger: lldb.SBDebugger) -> None:
    debugger.HandleCommand('command container add llgo')
    debugger.HandleCommand(
        'command script add -f llgo_plugin.print_target_status llgo status')
    debugger.HandleCommand(
        'command script add -f llgo_plugin.print_go_expression llgo print')
    debugger.HandleCommand(
        'command script add -f llgo_plugin.print_all_variables llgo vars')
    debugger.HandleCommand(
        'command script add -f llgo_plugin.print_goroutines llgo goroutines')
    debugger.HandleCommand(
        'command script add -f llgo_plugin.print_goroutine llgo goroutine')
    register_type_formatters(debugger)


def _type_options(hide_children: bool = False) -> int:
    options = lldb.eTypeOptionCascade
    options |= getattr(lldb, "eTypeOptionSkipPointers", 0)
    options |= getattr(lldb, "eTypeOptionSkipReferences", 0)
    if hide_children:
        options |= getattr(lldb, "eTypeOptionHideChildren", 0)
    return options


def register_type_formatters(debugger: lldb.SBDebugger) -> None:
    category = debugger.CreateCategory(LLGO_TYPE_CATEGORY)
    for layout in LLGO_RUNTIME_LAYOUTS.values():
        category.AddTypeSummary(
            lldb.SBTypeNameSpecifier(layout.string_type, False),
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.string_summary", _type_options(True)),
        )
        slice_specifier = lldb.SBTypeNameSpecifier(
            layout.slice_type_pattern, True)
        category.AddTypeSummary(
            slice_specifier,
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.slice_summary", _type_options()),
        )
        category.AddTypeSynthetic(
            slice_specifier,
            lldb.SBTypeSynthetic.CreateWithClassName(
                "llgo_plugin.SliceSyntheticProvider", _type_options()),
        )
        category.AddTypeSummary(
            lldb.SBTypeNameSpecifier(
                layout.interface_type_pattern, True),
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.interface_summary", _type_options()),
        )
        category.AddTypeSummary(
            lldb.SBTypeNameSpecifier(
                layout.function_type_pattern, True),
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.function_summary", _type_options()),
        )
        # LLDB spells a DWARF pointer as the pointee name followed by " *".
        descriptor = layout.function_descriptor_type_pattern
        if descriptor.endswith("$"):
            descriptor = descriptor[:-1]
        category.AddTypeSummary(
            lldb.SBTypeNameSpecifier(descriptor + r" \*$", True),
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.function_summary", _type_options()),
        )
        map_specifier = lldb.SBTypeNameSpecifier(
            layout.map_type_pattern, True)
        category.AddTypeSummary(
            map_specifier,
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.map_summary", _type_options()),
        )
        category.AddTypeSynthetic(
            map_specifier,
            lldb.SBTypeSynthetic.CreateWithClassName(
                "llgo_plugin.MapSyntheticProvider", _type_options()),
        )
        channel_specifier = lldb.SBTypeNameSpecifier(
            layout.channel_type_pattern, True)
        category.AddTypeSummary(
            channel_specifier,
            lldb.SBTypeSummary.CreateWithFunctionName(
                "llgo_plugin.channel_summary", _type_options()),
        )
        category.AddTypeSynthetic(
            channel_specifier,
            lldb.SBTypeSynthetic.CreateWithClassName(
                "llgo_plugin.ChannelSyntheticProvider", _type_options()),
        )
    category.SetEnabled(False)


def configure_target(debugger: lldb.SBDebugger) -> None:
    target = debugger.GetSelectedTarget()
    supported = inspect_target(target).supported
    debugger.GetCategory(LLGO_TYPE_CATEGORY).SetEnabled(
        supported)
    if supported and _local_windows_arm64_target(target):
        # Install before launch: unwound frames are cached, and changing the
        # address mask after a backtrace does not invalidate those frames.
        identity = target.GetGloballyUniqueID() if hasattr(target, "GetGloballyUniqueID") else None
        if identity is None or identity not in _CODE_ADDRESS_HOOK_TARGETS:
            result = lldb.SBCommandReturnObject()
            debugger.GetCommandInterpreter().HandleCommand(
                "target stop-hook add -P llgo_plugin.WindowsARM64CodeAddressHook", result)
            if result.Succeeded() and identity is not None:
                _CODE_ADDRESS_HOOK_TARGETS.add(identity)
        _configure_windows_arm64_code_addresses(target, target.GetProcess())


def _local_windows_arm64_target(target: lldb.SBTarget) -> bool:
    if sys.platform != "win32" or not target or not target.IsValid():
        return False
    try:
        triple = (target.GetTriple() or "").lower()
        return (triple.split("-", 1)[0] in ("arm64", "aarch64") and
                "windows" in triple.split("-") and
                target.GetAddressByteSize() == 8 and target.GetPlatform().IsHost())
    except (AttributeError, TypeError):
        # Older public APIs cannot establish whether this target is local.
        return False


def _windows_arm64_user_address_bits() -> Optional[int]:
    if sys.platform != "win32" or ctypes.sizeof(ctypes.c_void_p) != 8:
        return None

    class SystemInfo(ctypes.Structure):
        _fields_ = [
            ("processor", ctypes.c_uint32), ("page_size", ctypes.c_uint32),
            ("minimum", ctypes.c_void_p), ("maximum", ctypes.c_void_p),
            ("cpu_mask", ctypes.c_size_t), ("cpu_count", ctypes.c_uint32),
            ("cpu_type", ctypes.c_uint32), ("granularity", ctypes.c_uint32),
            ("cpu_level", ctypes.c_uint16), ("cpu_revision", ctypes.c_uint16),
        ]

    try:
        info = SystemInfo()
        get_info = ctypes.WinDLL("kernel32", use_last_error=True).GetSystemInfo
        get_info.argtypes = [ctypes.POINTER(SystemInfo)]
        get_info.restype = None
        get_info(ctypes.byref(info))
        # PROCESSOR_ARCHITECTURE_ARM64. A different or emulated host cannot
        # establish the native target's address bounds from this API.
        if ((info.processor & 0xffff) != 12 or not info.page_size or
                not info.minimum or not info.maximum or info.maximum < info.minimum):
            return None
        bits = info.maximum.bit_length()
        return bits if 32 < bits < 64 else None
    except (AttributeError, OSError, TypeError, ValueError):
        return None


def _configure_windows_arm64_code_addresses(target: lldb.SBTarget,
                                            process: lldb.SBProcess) -> None:
    try:
        if (not _local_windows_arm64_target(target) or not process or
                not process.IsValid() or process.GetPluginName() != "windows"):
            return
        code = lldb.eAddressMaskTypeCode
        # Zero is an explicit mask too. Respect both API masks and a user's
        # target.process.virtual-addressable-bits setting.
        if process.GetAddressMask(code) != lldb.LLDB_INVALID_ADDRESS_MASK:
            return
        bits = _windows_arm64_user_address_bits()
        if bits is not None:
            # LLDB 22 does not derive Windows ARM64 PAC masks. Use its public
            # code-address API so authenticated saved LRs can be unwound;
            # leave data addresses, target registers and memory unchanged.
            process.SetAddressableBits(code, bits)
    except (AttributeError, OSError, TypeError, ValueError):
        # Older debugger APIs or unavailable host information must not prevent
        # ordinary debugging. Never guess an address width for remote/dump targets.
        return


class WindowsARM64CodeAddressHook:
    def __init__(self, target: lldb.SBTarget, _args: Any,
                 _internal_dict: Any = None) -> None:
        self.target = target

    def handle_stop(self, exe_ctx: lldb.SBExecutionContext, _stream: Any) -> bool:
        if inspect_target(self.target).supported:
            _configure_windows_arm64_code_addresses(self.target, exe_ctx.GetProcess())
        return True


def _marker_versions(target: lldb.SBTarget) -> Tuple[int, ...]:
    if not target or not target.IsValid():
        return ()

    versions = set()
    marker_pattern = re.compile(
        rf"^{re.escape(LLGO_DEBUGGER_MARKER_PREFIX)}([0-9]+)$")
    for module_index in range(target.GetNumModules()):
        module = target.GetModuleAtIndex(module_index)
        for symbol_index in range(module.GetNumSymbols()):
            name = module.GetSymbolAtIndex(symbol_index).GetName()
            match = marker_pattern.match(name or "")
            if match:
                versions.add(int(match.group(1)))
    return tuple(sorted(versions))


def _sbdata_bytes(data: lldb.SBData, size: int) -> Optional[bytes]:
    if not data or not data.IsValid() or data.GetByteSize() < size:
        return None
    error = lldb.SBError()
    raw = bytes(data.GetUnsignedInt8(error, offset) for offset in range(size))
    return raw if error.Success() else None


def _read_debugger_record(target: lldb.SBTarget) -> Optional[bytes]:
    if not LLGO_DEBUGGER_RECORD_SYMBOL or LLGO_DEBUGGER_RECORD_SIZE <= 0:
        return None

    values = target.FindGlobalVariables(LLGO_DEBUGGER_RECORD_SYMBOL, 256)
    records = []
    if values and values.IsValid():
        for index in range(values.GetSize()):
            raw = _sbdata_bytes(
                values.GetValueAtIndex(index).GetData(),
                LLGO_DEBUGGER_RECORD_SIZE)
            if raw is not None:
                records.append(raw)

    if not records:
        for module_index in range(target.GetNumModules()):
            module = target.GetModuleAtIndex(module_index)
            for symbol_index in range(module.GetNumSymbols()):
                symbol = module.GetSymbolAtIndex(symbol_index)
                if symbol.GetName() != LLGO_DEBUGGER_RECORD_SYMBOL:
                    continue
                address = symbol.GetStartAddress()
                # Before launch there may be no readable process address. The
                # constant record is also available in the module's file data.
                section = address.GetSection()
                raw = _sbdata_bytes(section.GetSectionData(
                    address.GetOffset(), LLGO_DEBUGGER_RECORD_SIZE),
                    LLGO_DEBUGGER_RECORD_SIZE) if section.IsValid() else None
                if raw is None:
                    error = lldb.SBError()
                    raw = target.ReadMemory(address, LLGO_DEBUGGER_RECORD_SIZE, error)
                    if not error.Success():
                        raw = None
                if raw is not None:
                    records.append(bytes(raw))

    if not records:
        return None
    first = records[0]
    return first if all(record == first for record in records) else b""


def _record_field(raw: bytes, name: str) -> Optional[int]:
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


def _decode_debugger_record(
        raw: bytes) -> Tuple[Optional[LLGoDebuggerRecord], Optional[str]]:
    if len(raw) != LLGO_DEBUGGER_RECORD_SIZE:
        return None, "conflicting or incorrectly sized native records"
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
    return LLGoDebuggerRecord(
        record_version=values["record_version"],
        schema_version=values["schema_version"],
        runtime_layout_version=values["runtime_layout_version"],
        llgo_abi_version=values["llgo_abi_version"],
        pointer_size=values["pointer_size"],
        byte_order=values["byte_order"],
    ), None


def _byte_order_name(byte_order: int) -> str:
    return {
        lldb.eByteOrderBig: "big",
        lldb.eByteOrderPDP: "pdp",
        lldb.eByteOrderLittle: "little",
    }.get(byte_order, "unknown")


def _target_cache_key(target: lldb.SBTarget) -> Optional[Tuple[Any, ...]]:
    # File paths are not identities: a target can be rebuilt in place, and a
    # record unreadable before launch can become available in process memory.
    # Only reuse a successful inspection within one stopped process generation.
    try:
        process = target.GetProcess()
        if not process or not process.IsValid() or process.GetState() != lldb.eStateStopped:
            return None
        # A process belongs to one target. Its debugger-issued unique ID,
        # unlike the OS PID, changes on relaunch and distinguishes targets;
        # GetGloballyUniqueID is not available in current Apple SBTarget APIs.
        debugger_id, process_id = target.GetDebugger().GetID(), process.GetUniqueID()
        if not process_id:
            return None
        modules = []
        for index in range(target.GetNumModules()):
            module = target.GetModuleAtIndex(index)
            uuid = module.GetUUIDString()
            if not uuid:
                return None
            modules.append((uuid, str(module.GetFileSpec())))
        return (
            debugger_id, process_id, process.GetStopID(True),
            target.GetTriple() or "", target.GetAddressByteSize(),
            target.GetByteOrder(), tuple(modules),
        )
    except (AttributeError, TypeError):
        # Older LLDBs may lack an identity API. Uncached inspection is safe.
        return None


def inspect_target(target: lldb.SBTarget) -> LLGoTargetInfo:
    if not target or not target.IsValid():
        return LLGoTargetInfo((), None, None, "", 0, "unknown")

    cache_key = _target_cache_key(target)
    cached = _TARGET_INFO_CACHE.get(cache_key) if cache_key is not None else None
    if cached is not None:
        return cached
    # Bound the cache to the current target/process/stop and module set. A
    # launch, resume, stop or module change cannot reuse the previous result.
    _TARGET_INFO_CACHE.clear()

    marker_versions = _marker_versions(target)
    schema_version: Optional[int] = None
    runtime_layout_version: Optional[int] = None
    llgo_abi_version: Optional[int] = None
    record_version: Optional[int] = None
    compatibility_error: Optional[str] = None
    pointer_size = target.GetAddressByteSize()
    byte_order = _byte_order_name(target.GetByteOrder())

    raw_record = _read_debugger_record(target)
    if raw_record is not None:
        record, compatibility_error = _decode_debugger_record(raw_record)
        if record is not None:
            record_version = record.record_version
            schema_version = record.schema_version
            runtime_layout_version = record.runtime_layout_version
            llgo_abi_version = record.llgo_abi_version
            record_byte_order = LLGO_BYTE_ORDERS.get(record.byte_order)
            expected = (
                int(_RECORD_SCHEMA.get("version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get("schema_version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get("runtime_layout_version", 0)),
                int(LLGO_DEBUGGER_SCHEMA.get("llgo_abi_version", 0)),
            )
            actual = (
                record.record_version,
                record.schema_version,
                record.runtime_layout_version,
                record.llgo_abi_version,
            )
            if actual != expected:
                compatibility_error = (
                    "unsupported record/schema/runtime/ABI versions "
                    f"{actual}, want {expected}")
            elif record.pointer_size != pointer_size:
                compatibility_error = (
                    f"record pointer size {record.pointer_size} does not match "
                    f"target pointer size {pointer_size}")
            elif record_byte_order != byte_order:
                compatibility_error = (
                    f"record byte order {record_byte_order or record.byte_order} "
                    f"does not match target byte order {byte_order}")
            elif marker_versions and marker_versions != (record.schema_version,):
                compatibility_error = (
                    f"record schema v{record.schema_version} conflicts with "
                    f"legacy marker version(s) {marker_versions}")

    # Multiple markers are ambiguous: do not select a runtime layout merely
    # because one of the advertised schema versions happens to be supported.
    if raw_record is None and len(marker_versions) == 1:
        candidate = marker_versions[0]
        for (supported_schema, supported_runtime_layout,
             supported_llgo_abi) in (
                LLGO_DEBUGGER_SCHEMAS.values()):
            if candidate == supported_schema:
                schema_version = supported_schema
                runtime_layout_version = supported_runtime_layout
                llgo_abi_version = supported_llgo_abi
                break

    if ((marker_versions or raw_record is not None) and
            not LLGO_DEBUGGER_SCHEMA and compatibility_error is None):
        compatibility_error = (
            "debugger schema could not be loaded: " +
            (LLGO_DEBUGGER_SCHEMA_ERROR or "unknown error"))

    info = LLGoTargetInfo(
        marker_versions=marker_versions,
        schema_version=schema_version,
        runtime_layout_version=runtime_layout_version,
        triple=target.GetTriple() or "",
        pointer_size=pointer_size,
        byte_order=byte_order,
        record_version=record_version,
        llgo_abi_version=llgo_abi_version,
        compatibility_error=compatibility_error,
    )
    # Retry missing or incompatible records even in the same stop: an image
    # may have loaded before its debug information or memory became readable.
    if cache_key is not None and raw_record is not None and info.supported:
        _TARGET_INFO_CACHE[cache_key] = info
    return info


def target_status(info: LLGoTargetInfo) -> str:
    if not info.marker_versions and info.record_version is None:
        return "Not an LLGo target; raw LLDB debugging remains available."
    if info.compatibility_error:
        return (
            f"Unsupported LLGo debugger ABI: {info.compatibility_error}; "
            "raw LLDB debugging remains available."
        )
    if not info.supported:
        versions = ", ".join(f"v{version}"
                             for version in info.marker_versions)
        return (
            f"Unsupported LLGo debugger marker version(s): {versions}; "
            "raw LLDB debugging remains available."
        )
    abi = (f"LLGo ABI v{info.llgo_abi_version}; "
           if info.llgo_abi_version is not None else "")
    return (
        f"LLGo debugger schema v{info.schema_version} "
        f"(runtime layout v{info.runtime_layout_version}); "
        f"{abi}"
        f"target {info.triple}; pointer size {info.pointer_size}; "
        f"byte order {info.byte_order}."
    )


def is_llgo_compiler(target: lldb.SBTarget) -> bool:
    return inspect_target(target).supported


def print_target_status(debugger: lldb.SBDebugger, _command: str, result: lldb.SBCommandReturnObject, _internal_dict: Dict[str, Any]) -> None:
    result.AppendMessage(target_status(
        inspect_target(debugger.GetSelectedTarget())))


def _require_supported_target(debugger: lldb.SBDebugger, result: lldb.SBCommandReturnObject) -> bool:
    info = inspect_target(debugger.GetSelectedTarget())
    if info.supported:
        return True
    result.SetError(target_status(info))
    return False


def _stopped_process(debugger: lldb.SBDebugger, result: lldb.SBCommandReturnObject) -> Optional[lldb.SBProcess]:
    target = debugger.GetSelectedTarget()
    if not target or not target.IsValid():
        result.SetError("LLGo command requires a valid target.")
        return None

    process = target.GetProcess()
    if (not process or not process.IsValid() or
            process.GetState() != lldb.eStateStopped):
        result.SetError("LLGo command requires a stopped process.")
        return None
    _configure_windows_arm64_code_addresses(target, process)
    return process


def _selected_stopped_frame(debugger: lldb.SBDebugger, result: lldb.SBCommandReturnObject) -> Optional[lldb.SBFrame]:
    process = _stopped_process(debugger, result)
    if process is None:
        return None

    thread = process.GetSelectedThread()
    if not thread or not thread.IsValid():
        result.SetError("LLGo command requires a selected thread.")
        return None

    frame = thread.GetSelectedFrame()
    if not frame or not frame.IsValid():
        result.SetError("LLGo command requires a selected frame.")
        return None
    return frame


def _symbol_load_address(target: lldb.SBTarget, name: str) -> Optional[int]:
    contexts = target.FindSymbols(name)
    addresses = set()
    for index in range(contexts.GetSize()):
        symbol = contexts.GetContextAtIndex(index).GetSymbol()
        if not symbol or not symbol.IsValid() or symbol.GetName() != name:
            continue
        address = symbol.GetStartAddress().GetLoadAddress(target)
        if address != getattr(lldb, "LLDB_INVALID_ADDRESS", (1 << 64) - 1):
            addresses.add(address)
    if len(addresses) != 1:
        return None
    return next(iter(addresses))


def _required_integer_field(value: lldb.SBValue, name: str) -> int:
    field = value.GetChildMemberWithName(name)
    result = _value_as_int(field)
    if result is None:
        raise ValueError(f"cannot read runtime field {name!r}")
    return result


def _goroutine_values(target: lldb.SBTarget, process: lldb.SBProcess,
                      layout: LLGoGoroutineLayout) -> List[LLGoGoroutineValue]:
    head_address = _symbol_load_address(target, layout.head_symbol)
    if head_address is None:
        raise ValueError(
            "LLGo goroutine metadata is unavailable for this runtime.")

    error = lldb.SBError()
    address = process.ReadPointerFromMemory(head_address, error)
    if not error.Success():
        raise ValueError("cannot read the LLGo goroutine list")

    goroutine_type = target.FindFirstType(layout.goroutine_type)
    if not goroutine_type or not goroutine_type.IsValid():
        raise ValueError("LLGo goroutine type metadata is unavailable")

    status_names = dict(layout.status_names)
    visited = set()
    values: List[LLGoGoroutineValue] = []
    while address != 0:
        if address in visited:
            raise ValueError("LLGo goroutine list contains a cycle")
        if len(values) >= LLGO_MAX_GOROUTINES:
            raise ValueError("LLGo goroutine list exceeds the safety limit")
        visited.add(address)

        goroutine = target.CreateValueFromAddress(
            "__llgo_g", lldb.SBAddress(address, target), goroutine_type)
        if not goroutine or not goroutine.IsValid():
            raise ValueError("cannot read an LLGo goroutine")

        next_address = _required_integer_field(
            goroutine, layout.goroutine_next)
        status = _required_integer_field(goroutine, layout.goroutine_status)
        goid = _required_integer_field(goroutine, layout.goroutine_id)
        parent_goid = _required_integer_field(
            goroutine, layout.goroutine_parent_id)
        procid = _required_integer_field(goroutine, layout.procid)
        values.append(LLGoGoroutineValue(
            address=address,
            goid=goid,
            parent_goid=parent_goid,
            status=status,
            status_name=status_names.get(status, f"status-{status}"),
            procid=procid,
        ))
        address = next_address

    values.sort(key=lambda value: value.goid)
    return values


def _goroutine_thread(process: lldb.SBProcess,
                      goroutine: LLGoGoroutineValue) -> Optional[lldb.SBThread]:
    if goroutine.procid == 0:
        return None
    for index in range(process.GetNumThreads()):
        thread = process.GetThreadAtIndex(index)
        if (thread and thread.IsValid() and
                thread.GetThreadID() == goroutine.procid):
            return thread
    return None


def print_goroutines(debugger: lldb.SBDebugger, command: str,
                     result: lldb.SBCommandReturnObject,
                     _internal_dict: Dict[str, Any]) -> None:
    if not _require_supported_target(debugger, result):
        return
    if command.strip():
        result.SetError("usage: llgo goroutines")
        return
    process = _stopped_process(debugger, result)
    if process is None:
        return

    target = debugger.GetSelectedTarget()
    info = inspect_target(target)
    layout = LLGO_GOROUTINE_LAYOUTS.get(info.runtime_layout_version)
    if layout is None:
        result.SetError("LLGo goroutine metadata is unavailable for this runtime.")
        return
    try:
        goroutines = _goroutine_values(target, process, layout)
    except ValueError as error:
        result.SetError(str(error))
        return

    if not goroutines:
        result.AppendMessage("No live LLGo goroutines.")
        return
    lines = []
    for goroutine in goroutines:
        thread = _goroutine_thread(process, goroutine)
        thread_index = (str(thread.GetIndexID())
                        if thread is not None else "unavailable")
        lines.append(
            f"goroutine {goroutine.goid} [{goroutine.status_name}] "
            f"parent={goroutine.parent_goid} tid={goroutine.procid} "
            f"thread={thread_index}")
    result.AppendMessage("\n".join(lines))


def _frame_description(frame: lldb.SBFrame, index: int) -> str:
    name = frame.GetFunctionName() or frame.GetSymbol().GetName() or "<unknown>"
    description = f"frame #{index}: {name}"
    line_entry = frame.GetLineEntry()
    if line_entry and line_entry.IsValid():
        file_spec = line_entry.GetFileSpec()
        filename = file_spec.GetFilename() if file_spec else None
        line = line_entry.GetLine()
        if filename and line:
            description += f" at {filename}:{line}"
    return description


def print_goroutine(debugger: lldb.SBDebugger, command: str,
                    result: lldb.SBCommandReturnObject,
                    _internal_dict: Dict[str, Any]) -> None:
    if not _require_supported_target(debugger, result):
        return
    match = re.fullmatch(r"\s*([0-9]+)\s+(?:bt|backtrace)\s*", command)
    if match is None:
        result.SetError("usage: llgo goroutine <id> bt")
        return
    process = _stopped_process(debugger, result)
    if process is None:
        return

    target = debugger.GetSelectedTarget()
    info = inspect_target(target)
    layout = LLGO_GOROUTINE_LAYOUTS.get(info.runtime_layout_version)
    if layout is None:
        result.SetError("LLGo goroutine metadata is unavailable for this runtime.")
        return
    try:
        goroutines = _goroutine_values(target, process, layout)
    except ValueError as error:
        result.SetError(str(error))
        return

    goid = int(match.group(1))
    goroutine = next(
        (candidate for candidate in goroutines if candidate.goid == goid),
        None)
    if goroutine is None:
        result.SetError(f"LLGo goroutine {goid} is not live.")
        return
    thread = _goroutine_thread(process, goroutine)
    if thread is None:
        result.SetError(
            f"LLGo goroutine {goid} has no matching debugger thread.")
        return

    frame_count = min(thread.GetNumFrames(), LLGO_MAX_STACK_FRAMES)
    lines = [
        f"goroutine {goid} [{goroutine.status_name}] "
        f"thread {thread.GetIndexID()}:"
    ]
    lines.extend(_frame_description(thread.GetFrameAtIndex(index), index)
                 for index in range(frame_count))
    if thread.GetNumFrames() > frame_count:
        lines.append(
            f"... ({thread.GetNumFrames() - frame_count} more frames)")
    result.AppendMessage("\n".join(lines))


def _value_as_int(value: lldb.SBValue) -> Optional[int]:
    if not value or not value.IsValid():
        return None
    raw = value.GetValue()
    if raw is None:
        return None
    try:
        return int(raw, 0)
    except (TypeError, ValueError):
        error = value.GetError()
        if error and error.Fail():
            return None
        return value.GetValueAsUnsigned()


def _raw_value(value: lldb.SBValue) -> lldb.SBValue:
    if not value or not value.IsValid():
        return value
    raw = value.GetNonSyntheticValue()
    return raw if raw and raw.IsValid() else value


def _canonical_type_name(value: lldb.SBValue) -> str:
    value_type = _raw_value(value).GetType()
    while value_type and value_type.IsValid() and value_type.IsTypedefType():
        value_type = value_type.GetTypedefedType()
    return value_type.GetName() if value_type and value_type.IsValid() else ""


def _matches_type_pattern(value: lldb.SBValue, pattern: str) -> bool:
    value_type = _raw_value(value).GetType()
    while value_type and value_type.IsValid():
        if re.fullmatch(pattern, value_type.GetName() or ""):
            return True
        if not value_type.IsTypedefType():
            return False
        value_type = value_type.GetTypedefedType()
    return False


def _runtime_layout(value: lldb.SBValue) -> Optional[LLGoRuntimeLayout]:
    if not value or not value.IsValid():
        return None
    info = inspect_target(value.GetTarget())
    if not info.supported or info.runtime_layout_version is None:
        return None
    return LLGO_RUNTIME_LAYOUTS.get(info.runtime_layout_version)


def _string_fields(value: lldb.SBValue, layout: LLGoRuntimeLayout) -> Optional[Tuple[lldb.SBValue, int]]:
    raw = _raw_value(value)
    if _canonical_type_name(raw) != layout.string_type:
        return None
    data = raw.GetChildMemberWithName(layout.string_data)
    length = _value_as_int(raw.GetChildMemberWithName(layout.string_len))
    if not data or not data.IsValid() or length is None or length < 0:
        return None
    return data, length


def _slice_fields(value: lldb.SBValue,
                  layout: LLGoRuntimeLayout) -> Optional[LLGoSliceValue]:
    raw = _raw_value(value)
    if not re.fullmatch(layout.slice_type_pattern,
                        _canonical_type_name(raw)):
        return None
    data = raw.GetChildMemberWithName(layout.slice_data)
    length = _value_as_int(raw.GetChildMemberWithName(layout.slice_len))
    capacity = _value_as_int(raw.GetChildMemberWithName(layout.slice_cap))
    if (not data or not data.IsValid() or length is None or
            capacity is None or length < 0 or capacity < length):
        return None
    address = _value_as_int(data)
    element_type = data.GetType().GetPointeeType()
    if address is None:
        return None
    if not element_type or not element_type.IsValid():
        return None
    return LLGoSliceValue(
        address=address,
        length=length,
        capacity=capacity,
        element_type=element_type,
        element_size=element_type.GetByteSize(),
    )


def _slice_element(value: lldb.SBValue, index: int,
                   fields: LLGoSliceValue) -> Optional[lldb.SBValue]:
    if index < 0 or index >= fields.length or fields.address == 0:
        return None
    element_address = fields.address + index * fields.element_size
    target = value.GetTarget()
    return target.CreateValueFromAddress(
        f"[{index}]", lldb.SBAddress(element_address, target),
        fields.element_type)


def _quote_go_bytes(value: bytes) -> str:
    escapes = {
        "\a": r"\a",
        "\b": r"\b",
        "\f": r"\f",
        "\n": r"\n",
        "\r": r"\r",
        "\t": r"\t",
        "\v": r"\v",
        '"': r'\"',
        "\\": r"\\",
    }
    quoted: List[str] = ['"']
    for char in value.decode("utf-8", errors="surrogateescape"):
        escaped = escapes.get(char)
        if escaped is not None:
            quoted.append(escaped)
            continue
        code = ord(char)
        if 0xDC80 <= code <= 0xDCFF:
            quoted.append(f"\\x{code - 0xDC00:02x}")
        elif char.isprintable():
            quoted.append(char)
        elif code <= 0xFF:
            quoted.append(f"\\x{code:02x}")
        elif code <= 0xFFFF:
            quoted.append(f"\\u{code:04x}")
        else:
            quoted.append(f"\\U{code:08x}")
    quoted.append('"')
    return "".join(quoted)


def _utf8_bounded_prefix(value: bytes, limit: int) -> bytes:
    prefix = value[:limit]
    if len(value) <= limit or limit == 0:
        return prefix

    start = limit - 1
    lower_bound = max(0, limit - 4)
    while start >= lower_bound and value[start] & 0xC0 == 0x80:
        start -= 1
    if start < lower_bound:
        return prefix
    lead = value[start]
    if 0xC2 <= lead <= 0xDF:
        sequence_length = 2
    elif 0xE0 <= lead <= 0xEF:
        sequence_length = 3
    elif 0xF0 <= lead <= 0xF4:
        sequence_length = 4
    else:
        return prefix
    sequence_end = start + sequence_length
    if sequence_end <= limit or sequence_end > len(value):
        return prefix
    try:
        value[start:sequence_end].decode("utf-8")
    except UnicodeDecodeError:
        return prefix
    return value[:start]


def _format_runtime_string(value: lldb.SBValue,
                           layout: LLGoRuntimeLayout) -> Optional[str]:
    fields = _string_fields(value, layout)
    if fields is None:
        return None
    data, length = fields
    if length == 0:
        return '""'
    address = _value_as_int(data)
    process = value.GetProcess()
    if (address is None or address == 0 or not process or
            not process.IsValid()):
        return None
    display_length = min(length, LLGO_MAX_STRING_SUMMARY_BYTES)
    read_length = min(length, LLGO_MAX_STRING_SUMMARY_BYTES + 3)
    error = lldb.SBError()
    contents = process.ReadMemory(address, read_length, error)
    if not error.Success() or contents is None:
        return None
    if isinstance(contents, str):
        contents = contents.encode("latin-1", errors="surrogateescape")
    else:
        contents = bytes(contents)
    contents = _utf8_bounded_prefix(contents, display_length)
    summary = _quote_go_bytes(contents)
    return summary if display_length == length else summary + "..."


def string_summary(value: lldb.SBValue, _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    return _format_runtime_string(value, layout) if layout else None


def slice_summary(value: lldb.SBValue, _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    fields = _slice_fields(value, layout) if layout else None
    if fields is None:
        return None
    return f"len={fields.length} cap={fields.capacity}"


def _value_from_address(target: lldb.SBTarget, name: str, address: int,
                        type_name: str) -> Optional[lldb.SBValue]:
    value_type = target.FindFirstType(type_name)
    if not value_type or not value_type.IsValid():
        return None
    value = target.CreateValueFromAddress(
        name, lldb.SBAddress(address, target), value_type)
    return value if value and value.IsValid() else None


def _runtime_type_name(value: lldb.SBValue, address: int,
                       layout: LLGoRuntimeLayout) -> Optional[str]:
    runtime_type = _value_from_address(
        value.GetTarget(), "__llgo_runtime_type", address,
        layout.runtime_type)
    if runtime_type is None:
        return None

    name_value = runtime_type.GetChildMemberWithName(
        layout.runtime_type_string)
    fields = _string_fields(name_value, layout)
    if fields is None:
        return None
    data, length = fields
    if length > LLGO_MAX_TYPE_NAME_BYTES:
        return None
    data_address = _value_as_int(data)
    process = value.GetProcess()
    if (data_address is None or length < 0 or
            (length != 0 and data_address == 0) or not process or
            not process.IsValid()):
        return None

    error = lldb.SBError()
    contents = process.ReadMemory(data_address, length, error)
    if not error.Success() or contents is None:
        return None
    if isinstance(contents, str):
        contents = contents.encode("latin-1", errors="surrogateescape")
    try:
        name = bytes(contents).decode("utf-8")
    except UnicodeDecodeError:
        return None

    tflag = runtime_type.GetChildMemberWithName(layout.runtime_type_tflag)
    if (tflag and tflag.IsValid() and
            tflag.GetValueAsUnsigned(0) & layout.runtime_type_extra_star_flag):
        name = "*" + name
    return name


def _interface_type_address(value: lldb.SBValue,
                            layout: LLGoRuntimeLayout) -> Optional[int]:
    raw = _raw_value(value)
    if not re.fullmatch(layout.interface_type_pattern,
                        _canonical_type_name(raw)):
        return None
    type_word = raw.GetChildMemberWithName(layout.interface_type)
    data_word = raw.GetChildMemberWithName(layout.interface_data)
    if not data_word or not data_word.IsValid():
        return None
    type_address = _value_as_int(type_word)
    if type_address is None or type_address == 0:
        return type_address
    if _canonical_type_name(raw) == layout.empty_interface_type:
        return type_address

    itab = _value_from_address(
        value.GetTarget(), "__llgo_itab", type_address,
        layout.runtime_itab_type)
    if itab is None:
        return None
    return _value_as_int(itab.GetChildMemberWithName(
        layout.runtime_itab_concrete_type))


def interface_summary(value: lldb.SBValue,
                      _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    if layout is None:
        return None
    type_address = _interface_type_address(value, layout)
    if type_address is None:
        return None
    if type_address == 0:
        return "nil"
    type_name = _runtime_type_name(value, type_address, layout)
    return f"type={type_name}" if type_name else None


def _symbol_name(target: lldb.SBTarget, address: int) -> Optional[str]:
    if address == 0:
        return None
    resolved = target.ResolveLoadAddress(address)
    if not resolved or not resolved.IsValid():
        return None
    function = resolved.GetFunction()
    if function and function.IsValid():
        return function.GetName()
    symbol = resolved.GetSymbol()
    return symbol.GetName() if symbol and symbol.IsValid() else None


def function_summary(value: lldb.SBValue,
                     _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    raw = _raw_value(value)
    if layout is None:
        return None
    code_field, data_field = layout.function_code, layout.function_data
    pointee = raw.GetType().GetCanonicalType().GetPointeeType()
    pointee_name = pointee.GetName() or ""
    if pointee_name.startswith("struct "):
        pointee_name = pointee_name[len("struct "):]
    if re.fullmatch(layout.function_descriptor_type_pattern, pointee_name):
        address = _value_as_int(raw)
        if address is None:
            return None
        if address == 0:
            return "nil"
        raw = raw.Dereference()
        code_field, data_field = layout.function_descriptor_code, layout.function_descriptor_data
    elif not re.fullmatch(layout.function_type_pattern, _canonical_type_name(raw)):
        return None
    code = _value_as_int(raw.GetChildMemberWithName(code_field))
    data = _value_as_int(raw.GetChildMemberWithName(data_field))
    if code is None or data is None:
        return None
    if code == 0:
        return "nil"

    name = _symbol_name(value.GetTarget(), code)
    if name and name.startswith("__llgo_stub."):
        name = name[len("__llgo_stub."):]
    if not name:
        name = f"0x{code:x}"
    if name.endswith(layout.function_bound_symbol_suffix):
        return f"{name} (bound method)"
    if re.search(layout.function_closure_symbol_pattern, name):
        return f"{name} (closure)"
    return name


def _pointer_runtime_value(value: lldb.SBValue, pattern: str
                           ) -> Tuple[Optional[int], Optional[lldb.SBValue]]:
    raw = _raw_value(value)
    if not _matches_type_pattern(raw, pattern):
        return None, None
    address = _value_as_int(raw)
    if address is None or address == 0:
        return address, None
    pointee = raw.Dereference()
    if not pointee or not pointee.IsValid():
        return address, None
    pointee = pointee.GetNonSyntheticValue()
    return address, pointee if pointee and pointee.IsValid() else None


def map_summary(value: lldb.SBValue,
                _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    if layout is None:
        return None
    address, hash_value = _pointer_runtime_value(
        value, layout.map_type_pattern)
    if address is None:
        return None
    if address == 0:
        return "nil"
    if hash_value is None:
        return None
    length = _value_as_int(hash_value.GetChildMemberWithName(
        layout.map_count))
    return f"len={length}" if length is not None and length >= 0 else None


def _type_field(value_type: lldb.SBType, name: str) -> Optional[lldb.SBType]:
    while value_type and value_type.IsValid() and value_type.IsTypedefType():
        value_type = value_type.GetTypedefedType()
    if not value_type or not value_type.IsValid():
        return None
    for index in range(value_type.GetNumberOfFields()):
        field = value_type.GetFieldAtIndex(index)
        if field.GetName() == name:
            field_type = field.GetType()
            return field_type if field_type and field_type.IsValid() else None
    return None


def _channel_fields(value: lldb.SBValue,
                    layout: LLGoRuntimeLayout) -> Optional[LLGoChannelValue]:
    address, channel = _pointer_runtime_value(
        value, layout.channel_type_pattern)
    if address is None or address == 0 or channel is None:
        return None
    length = _value_as_int(channel.GetChildMemberWithName(
        layout.channel_count))
    capacity = _value_as_int(channel.GetChildMemberWithName(
        layout.channel_capacity))
    buffer = _value_as_int(channel.GetChildMemberWithName(
        layout.channel_buffer))
    receive_index = _value_as_int(channel.GetChildMemberWithName(
        layout.channel_receive_index))
    closed_value = channel.GetChildMemberWithName(layout.channel_closed)
    if (length is None or capacity is None or buffer is None or
            receive_index is None or length < 0 or capacity < length or
            (capacity != 0 and receive_index >= capacity) or
            not closed_value or not closed_value.IsValid()):
        return None

    channel_type = channel.GetType()
    queue_type = _type_field(channel_type, layout.channel_receive_queue)
    first_type = (_type_field(queue_type, layout.channel_queue_first)
                  if queue_type else None)
    waiter_type = (first_type.GetPointeeType()
                   if first_type and first_type.IsPointerType() else None)
    element_pointer = (_type_field(
        waiter_type, layout.channel_waiter_element)
        if waiter_type else None)
    element_type = (element_pointer.GetPointeeType()
                    if element_pointer and element_pointer.IsPointerType()
                    else None)
    if not element_type or not element_type.IsValid():
        return None
    element_size = element_type.GetByteSize()
    if element_size <= 0 or (length != 0 and buffer == 0):
        return None
    return LLGoChannelValue(
        length=length,
        capacity=capacity,
        buffer=buffer,
        receive_index=receive_index,
        closed=closed_value.GetValueAsUnsigned(0) != 0,
        element_type=element_type,
        element_size=element_size,
    )


def channel_summary(value: lldb.SBValue,
                    _internal_dict: Dict[str, Any]) -> Optional[str]:
    layout = _runtime_layout(value)
    if layout is None:
        return None
    address, _ = _pointer_runtime_value(value, layout.channel_type_pattern)
    if address is None:
        return None
    if address == 0:
        return "nil"
    fields = _channel_fields(value, layout)
    if fields is None:
        return None
    suffix = " closed" if fields.closed else ""
    return f"len={fields.length} cap={fields.capacity}{suffix}"


def _renamed_value(value: lldb.SBValue, name: str) -> Optional[lldb.SBValue]:
    if not value or not value.IsValid():
        return None
    renamed = value.Clone(name)
    return renamed if renamed and renamed.IsValid() else None


def _map_bucket_value(target: lldb.SBTarget, address: int,
                      bucket_type: lldb.SBType) -> Optional[lldb.SBValue]:
    if address == 0:
        return None
    bucket = target.CreateValueFromAddress(
        "__llgo_bucket", lldb.SBAddress(address, target), bucket_type)
    return bucket if bucket and bucket.IsValid() else None


def _map_bucket_evacuated(target: lldb.SBTarget, address: int,
                          bucket_type: lldb.SBType,
                          layout: LLGoRuntimeLayout) -> bool:
    bucket = _map_bucket_value(target, address, bucket_type)
    if bucket is None:
        return False
    tophash = bucket.GetChildMemberWithName(layout.map_bucket_tophash)
    first = tophash.GetChildAtIndex(0)
    value = _value_as_int(first)
    return (value is not None and
            layout.map_evacuated_tophash_min <= value <=
            layout.map_evacuated_tophash_max)


def _map_entries(value: lldb.SBValue, layout: LLGoRuntimeLayout,
                 max_entries: int) -> Optional[List[lldb.SBValue]]:
    _, hash_value = _pointer_runtime_value(value, layout.map_type_pattern)
    if hash_value is None:
        return []
    length = _value_as_int(hash_value.GetChildMemberWithName(
        layout.map_count))
    flags = _value_as_int(hash_value.GetChildMemberWithName(
        layout.map_flags))
    bucket_bits = _value_as_int(hash_value.GetChildMemberWithName(
        layout.map_bucket_bits))
    buckets = hash_value.GetChildMemberWithName(layout.map_buckets)
    old_buckets = hash_value.GetChildMemberWithName(layout.map_old_buckets)
    buckets_address = _value_as_int(buckets)
    old_buckets_address = _value_as_int(old_buckets)
    if (length is None or flags is None or bucket_bits is None or
            buckets_address is None or old_buckets_address is None or
            length < 0 or bucket_bits < 0 or bucket_bits >= 63):
        return None
    if length == 0:
        return []
    if buckets_address == 0:
        return None
    bucket_type = buckets.GetType().GetPointeeType()
    if not bucket_type or not bucket_type.IsValid():
        return None
    bucket_size = bucket_type.GetByteSize()
    if bucket_size <= 0:
        return None

    target = value.GetTarget()
    logical_buckets = 1 << bucket_bits
    scan_buckets = min(logical_buckets, LLGO_MAX_CONTAINER_SCAN_BUCKETS)
    old_count = (logical_buckets
                 if flags & layout.map_same_size_grow_flag
                 else logical_buckets >> 1)
    entries: List[lldb.SBValue] = []

    def append_chain(address: int) -> None:
        visited = set()
        while (address and address not in visited and
               len(entries) < max_entries * 2):
            visited.add(address)
            bucket = _map_bucket_value(target, address, bucket_type)
            if bucket is None:
                return
            tophash = bucket.GetChildMemberWithName(
                layout.map_bucket_tophash)
            keys = bucket.GetChildMemberWithName(layout.map_bucket_keys)
            indirect_keys = not keys or not keys.IsValid()
            if indirect_keys:
                keys = bucket.GetChildMemberWithName(
                    layout.map_bucket_indirect_keys)
            values = bucket.GetChildMemberWithName(
                layout.map_bucket_values)
            indirect_values = not values or not values.IsValid()
            if indirect_values:
                values = bucket.GetChildMemberWithName(
                    layout.map_bucket_indirect_values)
            if (not tophash.IsValid() or not keys.IsValid() or
                    not values.IsValid()):
                return
            slots = min(tophash.GetNumChildren(), keys.GetNumChildren(),
                        values.GetNumChildren())
            for slot in range(slots):
                top = _value_as_int(tophash.GetChildAtIndex(slot))
                if (top is None or
                        top < layout.map_occupied_tophash_min):
                    continue
                key = keys.GetChildAtIndex(slot)
                element = values.GetChildAtIndex(slot)
                if indirect_keys:
                    key = key.Dereference()
                if indirect_values:
                    element = element.Dereference()
                pair_index = len(entries) // 2
                key = _renamed_value(key, f"key[{pair_index}]")
                element = _renamed_value(element, f"value[{pair_index}]")
                if key is None or element is None:
                    return
                entries.extend((key, element))
                if len(entries) >= max_entries * 2:
                    return
            address = _value_as_int(bucket.GetChildMemberWithName(
                layout.map_bucket_overflow)) or 0

    for bucket_index in range(scan_buckets):
        bucket_address = buckets_address + bucket_index * bucket_size
        if old_buckets_address and old_count:
            old_index = bucket_index & (old_count - 1)
            old_address = old_buckets_address + old_index * bucket_size
            if not _map_bucket_evacuated(
                    target, old_address, bucket_type, layout):
                if bucket_index >= old_count:
                    continue
                bucket_address = old_address
        append_chain(bucket_address)
        if len(entries) >= min(length, max_entries) * 2:
            break
    return entries


class MapSyntheticProvider:
    def __init__(self, value: lldb.SBValue,
                 _internal_dict: Dict[str, Any]) -> None:
        self.value = value
        self.raw = _raw_value(value)
        self.layout = _runtime_layout(self.raw)
        self.entries: Optional[List[lldb.SBValue]] = None
        self.update()

    def update(self) -> bool:
        self.raw = _raw_value(self.value)
        self.entries = (_map_entries(
            self.raw, self.layout, LLGO_DEFAULT_MAX_CHILDREN // 2)
            if self.layout else None)
        return False

    def num_children(self, max_children: Optional[int] = None) -> int:
        count = (len(self.entries) if self.entries is not None
                 else self.raw.GetNumChildren())
        if max_children is not None and max_children >= 0:
            count = min(count, max_children)
        return count

    def get_child_at_index(self, index: int) -> Optional[lldb.SBValue]:
        if self.entries is None:
            return self.raw.GetChildAtIndex(index)
        return self.entries[index] if 0 <= index < len(self.entries) else None

    def get_child_index(self, name: str) -> int:
        if self.entries is None:
            return -1
        for index, child in enumerate(self.entries):
            if child.GetName() == name:
                return index
        return -1

    def has_children(self) -> bool:
        return self.num_children() != 0


class ChannelSyntheticProvider:
    def __init__(self, value: lldb.SBValue,
                 _internal_dict: Dict[str, Any]) -> None:
        self.value = value
        self.raw = _raw_value(value)
        self.layout = _runtime_layout(self.raw)
        self.fields: Optional[LLGoChannelValue] = None
        self.update()

    def update(self) -> bool:
        self.raw = _raw_value(self.value)
        self.fields = (_channel_fields(self.raw, self.layout)
                       if self.layout else None)
        return False

    def num_children(self, max_children: Optional[int] = None) -> int:
        count = (self.fields.length if self.fields is not None
                 else self.raw.GetNumChildren())
        if max_children is not None and max_children >= 0:
            count = min(count, max_children)
        return count

    def get_child_at_index(self, index: int) -> Optional[lldb.SBValue]:
        if self.fields is None:
            return self.raw.GetChildAtIndex(index)
        if (index < 0 or index >= self.fields.length or
                self.fields.capacity == 0 or self.fields.buffer == 0):
            return None
        buffer_index = (self.fields.receive_index + index) % self.fields.capacity
        address = self.fields.buffer + buffer_index * self.fields.element_size
        target = self.raw.GetTarget()
        return target.CreateValueFromAddress(
            f"[{index}]", lldb.SBAddress(address, target),
            self.fields.element_type)

    def get_child_index(self, name: str) -> int:
        if self.fields is None:
            return -1
        match = re.fullmatch(r"\[([0-9]+)\]", name or "")
        if match is None:
            return -1
        index = int(match.group(1))
        return index if index < self.num_children() else -1

    def has_children(self) -> bool:
        return self.num_children() != 0


class SliceSyntheticProvider:
    def __init__(self, value: lldb.SBValue, _internal_dict: Dict[str, Any]) -> None:
        self.value = value
        self.raw = _raw_value(value)
        self.layout = _runtime_layout(self.raw)
        self.fields: Optional[LLGoSliceValue] = None
        self.update()

    def update(self) -> bool:
        self.raw = _raw_value(self.value)
        self.fields = (_slice_fields(self.raw, self.layout)
                       if self.layout else None)
        return False

    def num_children(self, max_children: Optional[int] = None) -> int:
        if self.fields is None:
            count = self.raw.GetNumChildren()
        else:
            count = self.fields.length
        if max_children is not None and max_children >= 0:
            count = min(count, max_children)
        return count

    def get_child_at_index(self, index: int) -> Optional[lldb.SBValue]:
        if self.fields is None:
            return self.raw.GetChildAtIndex(index)
        return _slice_element(self.raw, index, self.fields)

    def get_child_index(self, name: str) -> int:
        if self.fields is None:
            for index in range(self.raw.GetNumChildren()):
                if self.raw.GetChildAtIndex(index).GetName() == name:
                    return index
            return -1
        match = re.fullmatch(r"\[([0-9]+)\]", name or "")
        if match is None:
            return -1
        index = int(match.group(1))
        return index if index < self.num_children() else -1

    def has_children(self) -> bool:
        return self.num_children() != 0


def get_indexed_value(value: lldb.SBValue, index: int) -> Optional[lldb.SBValue]:
    if not value or not value.IsValid():
        return None

    if value.GetType().IsArrayType():
        return value.GetChildAtIndex(index)
    layout = _runtime_layout(value)
    fields = _slice_fields(value, layout) if layout else None
    return _slice_element(value, index, fields) if fields else None


def find_variable(frame: lldb.SBFrame, name: str) -> lldb.SBValue:
    value = frame.FindVariable(name)
    if value and value.IsValid():
        return value
    target = frame.GetThread().GetProcess().GetTarget()
    return target.FindFirstGlobalVariable(name)


def evaluate_expression(frame: lldb.SBFrame, expression: str) -> Optional[lldb.SBValue]:
    parts = re.findall(r'\*|\w+|\(|\)|\[.*?\]|\.', expression)
    if not parts or "".join(parts) != re.sub(r"\s+", "", expression):
        return None

    def evaluate_part(i: int) -> Tuple[Optional[lldb.SBValue], int]:
        nonlocal parts
        value: Optional[lldb.SBValue] = None
        while i < len(parts):
            part = parts[i]

            if part == '*':
                sub_value, i = evaluate_part(i + 1)
                if sub_value and sub_value.IsValid():
                    value = sub_value.Dereference()
                else:
                    return None, i
            elif part == '(':
                depth = 1
                j = i + 1
                while j < len(parts) and depth > 0:
                    if parts[j] == '(':
                        depth += 1
                    elif parts[j] == ')':
                        depth -= 1
                    j += 1
                if depth != 0:
                    return None, j
                value, i = evaluate_part(i + 1)
                i = j - 1
            elif part == ')':
                return value, i + 1
            elif part == '.':
                if i + 1 >= len(parts) or not re.fullmatch(r'\w+', parts[i + 1]):
                    return None, i + 1
                if value is None:
                    value = find_variable(frame, parts[i+1])
                else:
                    value = value.GetChildMemberWithName(parts[i+1])
                i += 2
            elif part.startswith('['):
                try:
                    index = int(part[1:-1])
                except ValueError:
                    return None, i + 1
                value = get_indexed_value(value, index)
                i += 1
            else:
                if value is None:
                    value = find_variable(frame, part)
                else:
                    value = value.GetChildMemberWithName(part)
                i += 1

            if not value or not value.IsValid():
                return None, i

        return value, i

    value, _ = evaluate_part(0)
    return value


def print_go_expression(debugger: lldb.SBDebugger, command: str, result: lldb.SBCommandReturnObject, _internal_dict: Dict[str, Any]) -> None:
    if not _require_supported_target(debugger, result):
        return
    frame = _selected_stopped_frame(debugger, result)
    if frame is None:
        return
    value = evaluate_expression(frame, command)
    if value and value.IsValid():
        try:
            result.AppendMessage(format_value(value, debugger))
        except (IndexError, TypeError, ValueError) as error:
            result.SetError(f"Unable to format expression {command!r}: {error}")
    else:
        result.SetError(
            f"Error: Unable to evaluate expression '{command}'")


def print_all_variables(debugger: lldb.SBDebugger, _command: str, result: lldb.SBCommandReturnObject, _internal_dict: Dict[str, Any]) -> None:
    if not _require_supported_target(debugger, result):
        return

    frame = _selected_stopped_frame(debugger, result)
    if frame is None:
        return
    variables = frame.GetVariables(True, True, True, True)

    output: List[str] = []
    try:
        for var in variables:
            type_name = go_type_name(var.GetType())
            formatted = format_value(
                var, debugger, include_type=False, indent=0)
            output.append(f"var {var.GetName()} {type_name} = {formatted}")
    except (IndexError, TypeError, ValueError) as error:
        result.SetError(f"Unable to format LLGo variables: {error}")
        return

    result.AppendMessage("\n".join(output))


def is_pointer(frame: lldb.SBFrame, var_name: str) -> bool:
    var = find_variable(frame, var_name)
    return var.IsValid() and var.GetType().IsPointerType()


def format_value(var: lldb.SBValue, debugger: lldb.SBDebugger, include_type: bool = True, indent: int = 0) -> str:
    if not var.IsValid():
        return "<variable not available>"

    var_type = var.GetType()
    type_class = var_type.GetTypeClass()
    type_name = go_type_name(var_type)

    # Handle typedef types
    original_type_name = type_name
    while var_type.IsTypedefType():
        var_type = var_type.GetTypedefedType()
        type_name = go_type_name(var_type)
        type_class = var_type.GetTypeClass()

    if var_type.IsPointerType():
        layout = _runtime_layout(var)
        if (layout and
                (_matches_type_pattern(var, layout.map_type_pattern) or
                 _matches_type_pattern(var, layout.channel_type_pattern))):
            summary = var.GetSummary()
            if summary is not None:
                return summary
        return format_pointer(var, debugger, indent, original_type_name)

    if type_name.startswith('[]'):  # Slice
        return format_slice(var, debugger, indent)
    elif var_type.IsArrayType():
        return format_array(var, debugger, indent)
    elif type_name == 'string':  # String
        return format_string(var)
    elif type_class in [lldb.eTypeClassStruct, lldb.eTypeClassClass]:
        summary = var.GetSummary()
        if summary is not None:
            return summary
        return format_struct(var, debugger, include_type, indent, original_type_name)
    else:
        value = var.GetValue()
        summary = var.GetSummary()
        if value is not None:
            return f"{value}" if include_type else str(value)
        elif summary is not None:
            return f"{summary}" if include_type else summary
        else:
            return "<variable not available>"


def format_slice(var: lldb.SBValue, debugger: lldb.SBDebugger, indent: int) -> str:
    layout = _runtime_layout(var)
    fields = _slice_fields(var, layout) if layout else None
    if fields is None:
        return "<variable not available>"
    elements: List[str] = []

    indent_str = '  ' * indent
    next_indent_str = '  ' * (indent + 1)

    values = lldb.SBDebugger.GetInternalVariableValue(
        "target.max-children-count", debugger.GetInstanceName())
    max_children = LLGO_DEFAULT_MAX_CHILDREN
    if values.GetSize() != 0:
        try:
            max_children = max(0, int(values.GetStringAtIndex(0), 0))
        except (TypeError, ValueError):
            pass
    displayed = min(fields.length, max_children)
    for i in range(displayed):
        element = _slice_element(var, i, fields)
        if element is None or not element.IsValid():
            return "<variable not available>"
        value = format_value(
            element, debugger, include_type=False, indent=indent+1)
        elements.append(value)
    if displayed < fields.length:
        elements.append(f"... ({fields.length - displayed} more)")

    type_name = var.GetType().GetName()

    if len(elements) > 5:  # 如果元素数量大于5，则进行折行显示
        result = f"{type_name}{{\n{next_indent_str}" + \
            f",\n{next_indent_str}".join(elements) + f"\n{indent_str}}}"
    else:
        result = f"{type_name}{{{', '.join(elements)}}}"

    return result


def format_array(var: lldb.SBValue, debugger: lldb.SBDebugger, indent: int) -> str:
    elements: List[str] = []
    indent_str = '  ' * indent
    next_indent_str = '  ' * (indent + 1)

    for i in range(var.GetNumChildren()):
        value = format_value(var.GetChildAtIndex(
            i), debugger, include_type=False, indent=indent+1)
        elements.append(value)

    array_size = var.GetNumChildren()
    element_type = go_type_name(var.GetType().GetArrayElementType())
    type_name = f"[{array_size}]{element_type}"

    if len(elements) > 5:  # wrap line if too many elements
        return f"{type_name}{{\n{next_indent_str}" + f",\n{next_indent_str}".join(elements) + f"\n{indent_str}}}"
    else:
        return f"{type_name}{{{', '.join(elements)}}}"


def format_string(var: lldb.SBValue) -> str:
    layout = _runtime_layout(var)
    value = _format_runtime_string(var, layout) if layout else None
    return value if value is not None else "<variable not available>"


def format_struct(var: lldb.SBValue, debugger: lldb.SBDebugger, include_type: bool = True, indent: int = 0, type_name: str = "") -> str:
    children: List[str] = []
    indent_str = '  ' * indent
    next_indent_str = '  ' * (indent + 1)

    for i in range(var.GetNumChildren()):
        child = var.GetChildAtIndex(i)
        child_name = child.GetName()
        child_value = format_value(
            child, debugger, include_type=False, indent=indent+1)
        children.append(f"{child_name} = {child_value}")

    if len(children) > 5:  # 如果字段数量大于5，则进行折行显示
        struct_content = "{\n" + ",\n".join(
            [f"{next_indent_str}{child}" for child in children]) + f"\n{indent_str}}}"
    else:
        struct_content = f"{{{', '.join(children)}}}"

    if include_type:
        return f"{type_name}{struct_content}"
    else:
        return struct_content


def format_pointer(var: lldb.SBValue, _debugger: lldb.SBDebugger, _indent: int, _type_name: str) -> str:
    if not var.IsValid() or var.GetValueAsUnsigned() == 0:
        return "<variable not available>"
    return var.GetValue()  # Return the address as a string


TYPE_NAME_MAPPING: Dict[str, str] = {
    'long': 'int',
    'void': 'unsafe.Pointer',
    'char': 'byte',
    'short': 'int16',
    'int': 'int32',
    'long long': 'int64',
    'unsigned char': 'uint8',
    'unsigned short': 'uint16',
    'unsigned int': 'uint32',
    'unsigned long': 'uint',
    'unsigned long long': 'uint64',
    'float': 'float32',
    'double': 'float64',
}
TYPE_NAMES_BY_LENGTH = sorted(TYPE_NAME_MAPPING, key=len, reverse=True)


def map_type_name(type_name: str) -> str:
    # Handle pointer types
    if type_name.endswith('*'):
        base_type = type_name[:-1].strip()
        mapped_base_type = map_type_name(base_type)
        return f"*{mapped_base_type}"

    # Prefer longer C spellings so "unsigned long long" is considered before
    # its prefixes. The mapping is shared because this runs for every value in
    # recursive container formatting.
    for c_type in TYPE_NAMES_BY_LENGTH:
        if type_name.startswith(c_type):
            return type_name.replace(c_type, TYPE_NAME_MAPPING[c_type], 1)

    return type_name


def go_type_name(value_type: lldb.SBType) -> str:
    if value_type.IsPointerType():
        return f"*{go_type_name(value_type.GetPointeeType())}"
    if value_type.IsTypedefType():
        return value_type.GetName()
    return map_type_name(value_type.GetName())
