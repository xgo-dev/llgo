# pylint: disable=missing-module-docstring,missing-class-docstring,missing-function-docstring

import os
import sys
import argparse
import signal
from dataclasses import dataclass, field
from typing import List, Optional, Set, Dict, Any
import lldb
import llgo_plugin
from llgo_plugin import log


class LLDBTestException(Exception):
    pass


@dataclass
class Test:
    variable: str
    expected_value: str
    mode: str = "value"


@dataclass
class TestResult:
    test: Test
    status: str
    actual: Optional[str] = None
    message: Optional[str] = None
    missing: Optional[Set[str]] = None
    extra: Optional[Set[str]] = None


@dataclass
class TestCase:
    source_file: str
    marker: str
    tests: List[Test]


@dataclass
class CaseResult:
    test_case: TestCase
    function: str
    results: List[TestResult]


@dataclass
class TestResults:
    total: int = 0
    passed: int = 0
    failed: int = 0
    case_results: List[CaseResult] = field(default_factory=list)


def test_case(marker: str, expectations: List[tuple]) -> TestCase:
    return TestCase(
        source_file="main.go",
        marker=f"LLDB_BREAK: {marker}",
        tests=[Test(*expectation) for expectation in expectations],
    )


STRUCT_INITIAL = [
    ("all variables", "s"),
    ("s.i8", r"'\x01'"),
    ("s.i16", "2"),
    ("s.i32", "3"),
    ("s.i64", "4"),
    ("s.i", "5"),
    ("s.u8", r"'\x06'"),
    ("s.u16", "7"),
    ("s.u32", "8"),
    ("s.u64", "9"),
    ("s.u", "10"),
    ("s.f32", "11"),
    ("s.f64", "12"),
    ("s.b", "true"),
    ("s.c64", "13 + 14i"),
    ("s.c128", "15 + 16i"),
    ("s.slice", "[]int{21, 22, 23}"),
    ("s.arr", "[3]int{24, 25, 26}"),
    ("s.arr2", "[3]lldbtest.E{{i = 27}, {i = 28}, {i = 29}}"),
    ("s.s", '"hello"'),
    ("s.e", "lldbtest.E{i = 30}"),
    ("s.pad1", "100"),
    ("s.pad2", "200"),
]

STRUCT_VALUES_INITIAL = [
    ("all variables", "t s m b"),
    ("t.I", "1"),
    ("s.I", "2"),
    ("s.J", "3"),
    ("m.I", "4"),
    ("m.J", "5"),
    ("m.K", "6"),
    ("b.I", "7"),
    ("b.J", "8"),
    ("b.K", "9"),
    ("b.L", "10"),
    ("b.M", "11"),
    ("b.N", "12"),
    ("b.O", "13"),
    ("b.P", "14"),
    ("b.Q", "15"),
    ("b.R", "16"),
]

STRUCT_VALUES_UPDATED = [
    ("all variables", "t s m b"),
    ("t.I", "10"),
    ("s.I", "20"),
    ("s.J", "21"),
    ("m.I", "40"),
    ("m.J", "41"),
    ("m.K", "42"),
    ("b.I", "70"),
    ("b.J", "71"),
    ("b.K", "72"),
    ("b.L", "73"),
    ("b.M", "74"),
    ("b.N", "75"),
    ("b.O", "76"),
    ("b.P", "77"),
    ("b.Q", "78"),
    ("b.R", "79"),
]

TEST_CASES = [
    test_case("struct_param_initial", STRUCT_INITIAL),
    test_case("struct_param_updated", [
        ("s.i8", r"'\b'"),
        ("s.i16", "2"),
    ]),
    test_case("all_params_initial", [
        ("all variables", "i8 i16 i32 i64 i u8 u16 u32 u64 u f32 f64 b "
         "c64 c128 slice arr arr2 s e f pf pi intr m c err fn currentI32 "
         "currentI64 currentI currentU32 currentU64 currentU currentF32 currentF64"),
        ("i32", "3"),
        ("i64", "4"),
        ("i", "5"),
        ("u32", "8"),
        ("u64", "9"),
        ("u", "10"),
        ("f32", "11"),
        ("f64", "12"),
        ("currentI32", "3"),
        ("currentI64", "4"),
        ("currentI", "5"),
        ("currentU32", "8"),
        ("currentU64", "9"),
        ("currentU", "10"),
        ("currentF32", "11"),
        ("currentF64", "12"),
        ("slice", "[]int{21, 22, 23}"),
        ("arr", "[3]int{24, 25, 26}"),
        ("arr2", "[3]lldbtest.E{{i = 27}, {i = 28}, {i = 29}}"),
        ("slice[0]", "21"),
        ("slice[1]", "22"),
        ("slice[2]", "23"),
        ("arr[0]", "24"),
        ("arr[1]", "25"),
        ("arr[2]", "26"),
        ("arr2[0].i", "27"),
        ("arr2[1].i", "28"),
        ("arr2[2].i", "29"),
        ("e", "lldbtest.E{i = 30}"),
    ]),
    test_case("all_params_updated", [
        ("i8", r"'\t'"),
        ("i16", "10"),
        ("i32", "11"),
        ("i64", "12"),
        ("i", "13"),
        ("currentI32", "11"),
        ("currentI64", "12"),
        ("currentI", "13"),
        ("u8", r"'\x0e'"),
        ("u16", "15"),
        ("u32", "16"),
        ("u64", "17"),
        ("u", "18"),
        ("currentU32", "16"),
        ("currentU64", "17"),
        ("currentU", "18"),
        ("f32", "19"),
        ("f64", "20"),
        ("currentF32", "19"),
        ("currentF64", "20"),
        ("b", "false"),
        ("c64", "21 + 22i"),
        ("c128", "23 + 24i"),
        ("slice", "[]int{31, 32, 33}"),
        ("arr2", "[3]lldbtest.E{{i = 37}, {i = 38}, {i = 39}}"),
        ("s", '"world"'),
        ("e", "lldbtest.E{i = 40}"),
    ]),
    test_case("runtime_values", [
        ("all variables",
         "text empty binary unicodeText longUnicode invalid ints nilInts "
         "emptyInts namedText namedInts"),
        ("text", '"hello"'),
        ("empty", '""'),
        ("binary", r'"a\x00b"'),
        ("unicodeText", '"世界"'),
        ("invalid", r'"\xff"'),
        ("nilInts", "[]int{}"),
        ("emptyInts", "[]int{}"),
        ("namedText", '"named"'),
        ("namedInts", "lldbtest.NamedInts{11, 12, 13, 14}"),
        ("text", '"hello"', "summary"),
        ("empty", '""', "summary"),
        ("binary", r'"a\x00b"', "summary"),
        ("unicodeText", '"世界"', "summary"),
        ("longUnicode", '"' + "a" * 255 + '"...', "summary"),
        ("invalid", r'"\xff"', "summary"),
        ("namedText", '"named"', "summary"),
        ("ints", "len=2 cap=4", "summary"),
        ("nilInts", "len=0 cap=0", "summary"),
        ("emptyInts", "len=0 cap=0", "summary"),
        ("namedInts", "len=4 cap=4", "summary"),
        ("ints", "[0]=7, [1]=8", "synthetic"),
        ("nilInts", "", "synthetic"),
        ("emptyInts", "", "synthetic"),
        ("namedInts", "[0]=11, [1]=12, [2]=13, [3]=14", "synthetic"),
        ("ints", "[]int{7, ... (1 more)}", "limited"),
    ]),
    test_case("struct_values_initial", STRUCT_VALUES_INITIAL),
    test_case("struct_values_updated", STRUCT_VALUES_UPDATED),
    test_case("struct_ptrs_initial", STRUCT_VALUES_INITIAL),
    test_case("struct_ptrs_updated", STRUCT_VALUES_UPDATED),
    test_case("scope_if_entry", [
        ("all variables", "a branch"),
        ("a", "1"),
    ]),
    test_case("scope_if_true", [
        ("all variables", "a b c branch"),
        ("a", "1"),
        ("b", "2"),
        ("c", "3"),
        ("branch", "1"),
    ]),
    test_case("scope_if_false", [
        ("all variables", "a c d branch"),
        ("a", "1"),
        ("c", "3"),
        ("d", "4"),
        ("branch", "0"),
    ]),
    test_case("scope_if_exit", [
        ("all variables", "a branch"),
        ("a", "1"),
    ]),
    test_case("scope_for_zero", [
        ("all variables", "i a"),
        ("i", "0"),
        ("a", "1"),
    ]),
    test_case("scope_for_one", [
        ("all variables", "i a"),
        ("i", "1"),
        ("a", "1"),
    ]),
    test_case("scope_switch_one", [
        ("all variables", "i a b"),
        ("i", "1"),
        ("a", "0"),
        ("b", "1"),
    ]),
    test_case("scope_switch_two", [
        ("all variables", "i a c"),
        ("i", "2"),
        ("a", "0"),
        ("c", "2"),
    ]),
    test_case("scope_switch_default", [
        ("all variables", "i a d"),
        ("i", "3"),
        ("a", "0"),
        ("d", "3"),
    ]),
    test_case("scope_switch_exit", [
        ("all variables", "a i"),
        ("a", "0"),
    ]),
    test_case("main_struct_initial", [
        ("all variables", "s i err"),
        *STRUCT_INITIAL[1:-2],
        ("s.pf.i16", "100"),
        ("*(s.pf).i16", "100"),
        ("*(s.pi)", "100"),
    ]),
    test_case("main_globals", [
        ("all variables", "s i err"),
        ("globalInt", "301"),
        ("globalStruct.i8", r"'\x01'"),
        ("(*globalStructPtr).i16", "2"),
    ]),
    test_case("main_struct_updated", [
        ("all variables", "s i err"),
        ("s.i8", r"'\x12'"),
        ("(*globalStructPtr).i8", r"'\x12'"),
    ]),
]


class LLDBDebugger:
    def __init__(self, executable_path: str, plugin_path: Optional[str] = None) -> None:
        self.executable_path: str = executable_path
        self.plugin_path: Optional[str] = plugin_path
        self.debugger: lldb.SBDebugger = lldb.SBDebugger.Create()
        self.debugger.SetAsync(False)
        self.target: Optional[lldb.SBTarget] = None
        self.process: Optional[lldb.SBProcess] = None
        self.type_mapping: Dict[str, str] = {
            'long': 'int',
            'unsigned long': 'uint',
        }

    def setup(self) -> None:
        plugin_path = self.plugin_path or llgo_plugin.__file__
        self.debugger.HandleCommand(
            f'command script import "{plugin_path}"')
        self.target = self.debugger.CreateTarget(self.executable_path)
        if not self.target:
            raise LLDBTestException(
                f"Failed to create target for {self.executable_path}")

        target_info = llgo_plugin.inspect_target(self.target)
        if not target_info.supported:
            raise LLDBTestException(
                "Target does not contain a supported LLGo debugger marker")
        if llgo_plugin.inspect_target(self.target) is not target_info:
            raise LLDBTestException("LLGo target inspection was not cached")
        if (target_info.schema_version != 1 or
                target_info.runtime_layout_version != 1):
            raise LLDBTestException(
                f"Unexpected LLGo debugger schema: {target_info}")
        if (target_info.pointer_size != self.target.GetAddressByteSize() or
                target_info.byte_order == "unknown" or
                not target_info.triple):
            raise LLDBTestException(
                f"Incomplete LLGo target properties: {target_info}")

    def set_breakpoint(self, file_spec: str, line_number: int) -> lldb.SBBreakpoint:
        bp = self.target.BreakpointCreateByLocation(file_spec, line_number)
        if not bp.IsValid() or bp.GetNumLocations() != 1:
            raise LLDBTestException(
                f"Expected one breakpoint at {file_spec}:{line_number}, "
                f"found {bp.GetNumLocations()}")
        return bp

    def run_to_breakpoint(self) -> None:
        if not self.process:
            self.process = self.target.LaunchSimple(None, None, os.getcwd())
        else:
            self.process.Continue()
        if self.process.GetState() != lldb.eStateStopped:
            raise LLDBTestException("Process didn't stop at breakpoint")

    def get_variable_value(self, var_expression: str) -> Optional[str]:
        value = self.get_variable(var_expression)
        if value and value.IsValid():
            return llgo_plugin.format_value(value, self.debugger)
        return None

    def get_variable(self, var_expression: str) -> Optional[lldb.SBValue]:
        frame = self.process.GetSelectedThread().GetFrameAtIndex(0)
        return llgo_plugin.evaluate_expression(frame, var_expression)

    def get_variable_summary(self, var_expression: str) -> Optional[str]:
        value = self.get_variable(var_expression)
        if value and value.IsValid():
            return value.GetSummary()
        return None

    def get_synthetic_children(self, var_expression: str) -> Optional[str]:
        value = self.get_variable(var_expression)
        if not value or not value.IsValid():
            return None
        value = value.GetSyntheticValue()
        if not value or not value.IsValid():
            return None
        children: List[str] = []
        for index in range(value.GetNumChildren()):
            child = value.GetChildAtIndex(index)
            child_value = llgo_plugin.format_value(
                child, self.debugger, include_type=False)
            children.append(f"{child.GetName()}={child_value}")
        return ", ".join(children)

    def get_all_variable_names(self) -> Set[str]:
        frame = self.process.GetSelectedThread().GetFrameAtIndex(0)
        return set(var.GetName() for var in frame.GetVariables(True, True, False, True))

    def get_current_function_name(self) -> str:
        frame = self.process.GetSelectedThread().GetFrameAtIndex(0)
        return frame.GetFunctionName()

    def require_print_error(self, expression: str, expected: str) -> None:
        result = lldb.SBCommandReturnObject()
        llgo_plugin.print_go_expression(
            self.debugger, expression, result, {})
        if result.Succeeded() or expected not in (result.GetError() or ""):
            raise LLDBTestException(
                f"llgo print {expression!r} did not fail with {expected!r}: "
                f"{result.GetOutput()!r} {result.GetError()!r}")

    def cleanup(self) -> None:
        if self.process and self.process.IsValid():
            self.process.Kill()
        lldb.SBDebugger.Destroy(self.debugger)

    def run_console(self) -> bool:
        log("\nEntering LLDB interactive mode.")
        log("Type 'quit' to exit and continue with the next test case.")
        log("Use Ctrl+D to exit and continue, or Ctrl+C to abort all tests.")

        old_stdin, old_stdout, old_stderr = sys.stdin, sys.stdout, sys.stderr
        sys.stdin, sys.stdout, sys.stderr = sys.__stdin__, sys.__stdout__, sys.__stderr__

        self.debugger.SetAsync(True)
        self.debugger.HandleCommand("settings set auto-confirm true")
        self.debugger.HandleCommand("command script import lldb")

        interpreter = self.debugger.GetCommandInterpreter()
        continue_tests = True

        def keyboard_interrupt_handler(_sig: Any, _frame: Any) -> None:
            nonlocal continue_tests
            log("\nTest execution aborted by user.")
            continue_tests = False
            raise KeyboardInterrupt

        original_handler = signal.signal(
            signal.SIGINT, keyboard_interrupt_handler)

        try:
            while continue_tests:
                log("\n(lldb) ", end="")
                try:
                    command = input().strip()
                except EOFError:
                    log("\nExiting LLDB interactive mode. Continuing with next test case.")
                    break
                except KeyboardInterrupt:
                    break

                if command.lower() == 'quit':
                    log("\nExiting LLDB interactive mode. Continuing with next test case.")
                    break

                result = lldb.SBCommandReturnObject()
                interpreter.HandleCommand(command, result)
                log(result.GetOutput().rstrip() if result.Succeeded()
                    else result.GetError().rstrip())

        finally:
            signal.signal(signal.SIGINT, original_handler)
            sys.stdin, sys.stdout, sys.stderr = old_stdin, old_stdout, old_stderr

        return continue_tests


def marker_line(source_file: str, marker: str) -> int:
    with open(source_file, 'r', encoding='utf-8') as source:
        matches = [
            line_number for line_number, line in enumerate(source, start=1)
            if marker in line
        ]
    if len(matches) != 1:
        raise LLDBTestException(
            f"Expected one {marker!r} marker in {source_file}, "
            f"found {len(matches)}")
    return matches[0]


def execute_tests(executable_path: str, test_cases: List[TestCase], verbose: bool, interactive: bool, plugin_path: Optional[str]) -> TestResults:
    results = TestResults()

    for test_case in test_cases:
        debugger = LLDBDebugger(executable_path, plugin_path)
        try:
            line_number = marker_line(
                test_case.source_file, test_case.marker)
            if verbose:
                log(
                    f"\nSetting breakpoint at {test_case.source_file}:"
                    f"{line_number} ({test_case.marker})")
            debugger.setup()
            debugger.set_breakpoint(test_case.source_file, line_number)
            debugger.run_to_breakpoint()
            if not results.case_results:
                debugger.require_print_error(
                    "slice[bad]", "Unable to evaluate expression")

            all_variable_names = debugger.get_all_variable_names()

            case_result = execute_test_case(
                debugger, test_case, all_variable_names)

            results.total += len(case_result.results)
            results.passed += sum(1 for r in case_result.results if r.status == 'pass')
            results.failed += sum(1 for r in case_result.results if r.status != 'pass')
            results.case_results.append(case_result)

            case = case_result.test_case
            loc = f"{case.source_file}:{line_number} ({case.marker})"
            if verbose or interactive or any(r.status != 'pass' for r in case_result.results):
                log(f"\nTest case: {loc} in function '{case_result.function}'")
            for result in case_result.results:
                print_test_result(result, verbose=verbose)

            if interactive and any(r.status != 'pass' for r in case_result.results):
                log("\nTest case failed. Entering LLDB interactive mode.")
                continue_tests = debugger.run_console()
                if not continue_tests:
                    log("Aborting all tests.")
                    break

        finally:
            debugger.cleanup()

    return results


def run_tests(executable_path: str, source_files: List[str], verbose: bool, interactive: bool, plugin_path: Optional[str]) -> int:
    selected_sources = {os.path.basename(path) for path in source_files}
    test_cases = [
        test_case for test_case in TEST_CASES
        if test_case.source_file in selected_sources
    ]
    if not test_cases:
        raise LLDBTestException(
            f"No test cases registered for {', '.join(source_files)}")
    if verbose:
        log(f"Running tests for {', '.join(source_files)} with {executable_path}")
        log(f"Found {len(test_cases)} test cases")

    results = execute_tests(executable_path, test_cases,
                            verbose, interactive, plugin_path)
    print_test_results(results)

    # Return 0 if all tests passed, 1 otherwise
    return 0 if results.failed == 0 else 1


def execute_test_case(debugger: LLDBDebugger, test_case: TestCase, all_variable_names: Set[str]) -> CaseResult:
    results: List[TestResult] = []

    for test in test_case.tests:
        if test.variable == "all variables":
            result = execute_all_variables_test(test, all_variable_names)
        else:
            result = execute_single_variable_test(debugger, test)
        results.append(result)

    return CaseResult(test_case, debugger.get_current_function_name(), results)


def execute_all_variables_test(test: Test, all_variable_names: Set[str]) -> TestResult:
    expected_vars = set(test.expected_value.split())
    if expected_vars == all_variable_names:
        return TestResult(
            test=test,
            status='pass',
            actual=all_variable_names
        )
    else:
        return TestResult(
            test=test,
            status='fail',
            actual=all_variable_names,
            missing=expected_vars - all_variable_names,
            extra=all_variable_names - expected_vars
        )


def execute_single_variable_test(debugger: LLDBDebugger, test: Test) -> TestResult:
    if test.mode == "summary":
        actual_value = debugger.get_variable_summary(test.variable)
    elif test.mode == "synthetic":
        actual_value = debugger.get_synthetic_children(test.variable)
    elif test.mode == "limited":
        debugger.debugger.HandleCommand(
            "settings set target.max-children-count 1")
        try:
            actual_value = debugger.get_variable_value(test.variable)
        finally:
            debugger.debugger.HandleCommand(
                "settings set target.max-children-count 256")
    else:
        actual_value = debugger.get_variable_value(test.variable)
    if actual_value is None:
        return TestResult(
            test=test,
            status='error',
            message=f'Unable to fetch value for {test.variable}'
        )

    actual_value = actual_value.strip()
    expected_value = test.expected_value.strip()

    if actual_value == expected_value:
        return TestResult(
            test=test,
            status='pass',
            actual=actual_value
        )
    else:
        return TestResult(
            test=test,
            status='fail',
            actual=actual_value
        )


def print_test_results(results: TestResults) -> None:
    log("\nTest results:")
    log(f"  Total tests: {results.total}")
    log(f"  Passed tests: {results.passed}")
    log(f"  Failed tests: {results.failed}")
    if results.total == results.passed:
        log("All tests passed!")
    else:
        log("Some tests failed")


def print_test_result(result: TestResult, verbose: bool) -> None:
    status_symbol = "✓" if result.status == 'pass' else "✗"
    status_text = "Pass" if result.status == 'pass' else "Fail"
    test = result.test

    if result.status == 'pass':
        if verbose:
            log(f"{status_symbol} {test.variable}: {status_text}")
            if test.variable == 'all variables':
                log(f"    Variables: {', '.join(sorted(result.actual))}")
    else:  # fail or error
        log(f"{status_symbol} {test.variable}: {status_text}")
        if test.variable == 'all variables':
            if result.missing:
                log(f"    Missing variables: {', '.join(sorted(result.missing))}")
            if result.extra:
                log(f"    Extra variables: {', '.join(sorted(result.extra))}")
            log(f"    Expected: {', '.join(sorted(test.expected_value.split()))}")
            log(f"    Actual: {', '.join(sorted(result.actual))}")
        elif result.status == 'error':
            log(f"    Error: {result.message}")
        else:
            log(f"    Expected: {test.expected_value}")
            log(f"    Actual: {result.actual}")


def run_tests_with_result(executable_path: str, source_files: List[str], verbose: bool, interactive: bool, plugin_path: Optional[str], result_path: str) -> int:
    try:
        exit_code = run_tests(executable_path, source_files,
                              verbose, interactive, plugin_path)
    except Exception as e:
        log(f"An error occurred during test execution: {str(e)}")
        exit_code = 2  # Use a different exit code for unexpected errors

    try:
        with open(result_path, 'w', encoding='utf-8') as f:
            f.write(str(exit_code))
    except IOError as e:
        log(f"Error writing result to file {result_path}: {str(e)}")
        # If we can't write to the file, we should still return the exit code

    return exit_code


def main() -> None:
    log(sys.argv)
    parser = argparse.ArgumentParser(
        description="LLDB 18 Debug Script with DWARF 5 Support")
    parser.add_argument("executable", help="Path to the executable")
    parser.add_argument("sources", nargs='+', help="Paths to the source files")
    parser.add_argument("-v", "--verbose", action="store_true",
                        help="Enable verbose output")
    parser.add_argument("-i", "--interactive", action="store_true",
                        help="Enable interactive mode on test failure")
    parser.add_argument("--plugin", help="Path to the LLDB plugin")
    parser.add_argument("--result-path", help="Path to write the result")
    args = parser.parse_args()

    plugin_path = args.plugin

    try:
        if args.result_path:
            exit_code = run_tests_with_result(args.executable, args.sources,
                                              args.verbose, args.interactive, plugin_path, args.result_path)
        else:
            exit_code = run_tests(args.executable, args.sources,
                                  args.verbose, args.interactive, plugin_path)
    except Exception as e:
        log(f"An unexpected error occurred: {str(e)}")
        exit_code = 2  # Use a different exit code for unexpected errors

    sys.exit(exit_code)


if __name__ == "__main__":
    main()
