#!/bin/bash
# Build cache verification test script
# This script tests various cache scenarios and compares output against snapshots

set -o pipefail  # Make pipelines return failure if any command fails

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SNAPSHOTS_DIR="$SCRIPT_DIR/snapshots"

# Create temp directory for build outputs
get_temp_dir() {
    if [ -n "$TMPDIR" ]; then
        echo "$TMPDIR"
    elif [ -d "/tmp" ]; then
        echo "/tmp"
    else
        echo "."
    fi
}

TEMP_BASE="$(get_temp_dir)"
BUILD_TEMP_DIR="$TEMP_BASE/llgo-cache-test-$$"
mkdir -p "$BUILD_TEMP_DIR"

# Cleanup function
cleanup() {
    rm -rf "$BUILD_TEMP_DIR"
}
trap cleanup EXIT

# Determine cache directory based on platform
get_cache_dir() {
    case "$(uname -s)" in
        Darwin)
            echo "$HOME/Library/Caches/llgo/build"
            ;;
        Linux)
            echo "${XDG_CACHE_HOME:-$HOME/.cache}/llgo/build"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "$LOCALAPPDATA/llgo/build"
            ;;
        *)
            echo "$HOME/.cache/llgo/build"
            ;;
    esac
}

CACHE_DIR="$(get_cache_dir)"

# Helper function to clear cache
clear_cache() {
    echo "Clearing cache: $CACHE_DIR"
    rm -rf "$CACHE_DIR"
}

# Helper function to extract and sort cache status for buildcache packages only
extract_cache_status() {
    grep "^CACHE" | grep "buildcache" | sort
}

# Helper function to compare with snapshot
compare_snapshot() {
    local test_name="$1"
    local snapshot_file="$2"
    local actual_file="$3"

    if diff -q "$snapshot_file" "$actual_file" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        echo "Expected (snapshot):"
        cat "$snapshot_file"
        echo "Actual:"
        cat "$actual_file"
        echo "Diff:"
        diff "$snapshot_file" "$actual_file" || true
        return 1
    fi
}

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    shift
    echo -e "${YELLOW}Running: $test_name${NC}"
    if "$@"; then
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Test suite runner
run_test_suite() {
    local mode="$1"  # "native", "wasm", or "esp32c3"
    local build_cmd="$2"
    local output_file="$3"
    local run_cmd="$4"  # optional, for wasm

    echo ""
    echo "=========================================="
    case "$mode" in
        native)  echo "Native Build Cache Tests" ;;
        wasm)    echo "WASM Build Cache Tests" ;;
        esp32c3) echo "ESP32-C3 Build Cache Tests" ;;
        *)       echo "$mode Build Cache Tests" ;;
    esac
    echo "=========================================="

    # ===========================================================
    # TEST 1: First build - all CACHE MISS
    # ===========================================================
    test_01_first_build() {
        clear_cache
        if ! eval "$build_cmd" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_01.txt"; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi
        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi
        compare_snapshot "01_first_build ($mode)" "$SNAPSHOTS_DIR/01_first_build.txt" "$BUILD_TEMP_DIR/cache_${mode}_01.txt"
    }
    run_test "Test 01: First build (all CACHE MISS) [$mode]" test_01_first_build

    # ===========================================================
    # TEST 2: Second build - dependencies CACHE HIT
    # ===========================================================
    test_02_second_build() {
        if ! eval "$build_cmd" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_02.txt"; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi
        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi
        compare_snapshot "02_second_build_cache_hit ($mode)" "$SNAPSHOTS_DIR/02_second_build_cache_hit.txt" "$BUILD_TEMP_DIR/cache_${mode}_02.txt"
    }
    run_test "Test 02: Second build (deps CACHE HIT) [$mode]" test_02_second_build

    # ===========================================================
    # TEST 3: Force rebuild with -a flag - all CACHE MISS
    # ===========================================================
    test_03_force_rebuild() {
        if ! eval "${build_cmd/build/build -a}" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_03.txt"; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi
        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi
        compare_snapshot "03_force_rebuild ($mode)" "$SNAPSHOTS_DIR/03_force_rebuild.txt" "$BUILD_TEMP_DIR/cache_${mode}_03.txt"
    }
    run_test "Test 03: Force rebuild with -a (all CACHE MISS) [$mode]" test_03_force_rebuild

    # ===========================================================
    # TEST 4: Dependency change invalidates cache
    # ===========================================================
    test_04_dep1_changed() {
        # First ensure cache is populated
        if ! eval "$build_cmd" > /dev/null 2>&1; then
            echo -e "${RED}Initial build failed${NC}"
            return 1
        fi

        # Modify dep1
        echo "// modified for test" >> dep1/dep1.go

        # Build and capture output
        local build_result=0
        if ! eval "$build_cmd" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_04.txt"; then
            build_result=1
        fi

        # Restore dep1 (always do this, even if build failed)
        sed -i.bak '/^\/\/ modified for test$/d' dep1/dep1.go && rm -f dep1/dep1.go.bak

        if [ $build_result -ne 0 ]; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi

        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi

        compare_snapshot "04_dep1_changed ($mode)" "$SNAPSHOTS_DIR/04_dep1_changed.txt" "$BUILD_TEMP_DIR/cache_${mode}_04.txt"
    }
    run_test "Test 04: Dependency change invalidates cache [$mode]" test_04_dep1_changed

    # ===========================================================
    # TEST 5: Partial cache clear - only dep2
    # ===========================================================
    test_05_partial_clear_dep2() {
        # Rebuild to populate cache
        if ! eval "${build_cmd/build/build -a}" > /dev/null 2>&1; then
            echo -e "${RED}Initial build failed${NC}"
            return 1
        fi

        # Clear only dep2 cache
        find "$CACHE_DIR" -type d -name "*dep2*" -exec rm -rf {} + 2>/dev/null || true

        # Build and capture output
        if ! eval "$build_cmd" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_05.txt"; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi
        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi
        compare_snapshot "05_partial_clear_dep2 ($mode)" "$SNAPSHOTS_DIR/05_partial_clear_dep2.txt" "$BUILD_TEMP_DIR/cache_${mode}_05.txt"
    }
    run_test "Test 05: Partial cache clear (dep2 only) [$mode]" test_05_partial_clear_dep2

    # ===========================================================
    # TEST 6: Partial cache clear - dep2 and dep3
    # ===========================================================
    test_06_partial_clear_dep2_dep3() {
        # Rebuild to populate cache
        if ! eval "${build_cmd/build/build -a}" > /dev/null 2>&1; then
            echo -e "${RED}Initial build failed${NC}"
            return 1
        fi

        # Clear dep2 and dep3 cache
        find "$CACHE_DIR" -type d -name "*dep2*" -exec rm -rf {} + 2>/dev/null || true
        find "$CACHE_DIR" -type d -name "*dep3*" -exec rm -rf {} + 2>/dev/null || true

        # Build and capture output
        if ! eval "$build_cmd" 2>&1 | extract_cache_status > "$BUILD_TEMP_DIR/cache_${mode}_06.txt"; then
            echo -e "${RED}Build failed${NC}"
            return 1
        fi
        if [ -n "$run_cmd" ]; then
            if ! eval "$run_cmd" > /dev/null 2>&1; then
                echo -e "${RED}Run failed${NC}"
                return 1
            fi
        fi
        compare_snapshot "06_partial_clear_dep2_dep3 ($mode)" "$SNAPSHOTS_DIR/06_partial_clear_dep2_dep3.txt" "$BUILD_TEMP_DIR/cache_${mode}_06.txt"
    }
    run_test "Test 06: Partial cache clear (dep2 and dep3) [$mode]" test_06_partial_clear_dep2_dep3
}

# Change to test directory
cd "$SCRIPT_DIR"

echo "=========================================="
echo "Build Cache Verification Tests"
echo "=========================================="
echo "Cache directory: $CACHE_DIR"
echo "Snapshots directory: $SNAPSHOTS_DIR"
echo "Build temp directory: $BUILD_TEMP_DIR"

# Run native tests
run_test_suite "native" "llgo build -o $BUILD_TEMP_DIR/buildcache.out -debug ." ""

# Run WASM tests - always use iwasm from llgo cache directory
# Determine cache directory based on platform
if [ "$(uname -s)" = "Darwin" ]; then
    LLGO_IWASM_DIR="$HOME/Library/Caches/llgo/bin"
else
    LLGO_IWASM_DIR="${XDG_CACHE_HOME:-$HOME/.cache}/llgo/bin"
fi

LLGO_IWASM="$LLGO_IWASM_DIR/iwasm"

# Build iwasm if it doesn't exist in llgo cache
if [ ! -f "$LLGO_IWASM" ]; then
    echo ""
    echo -e "${YELLOW}iwasm not found in llgo cache, building it...${NC}"
    if [ -f "$SCRIPT_DIR/../../dev/build_iwasm.sh" ]; then
        bash "$SCRIPT_DIR/../../dev/build_iwasm.sh"
    else
        echo -e "${RED}Error: dev/build_iwasm.sh not found${NC}"
        echo -e "${BLUE}Skipping WASM tests${NC}"
        LLGO_IWASM=""
    fi
fi

if [ -n "$LLGO_IWASM" ] && [ -f "$LLGO_IWASM" ]; then
    echo ""
    echo -e "${BLUE}Using iwasm: $LLGO_IWASM${NC}"
    run_test_suite "wasm" \
        "GOOS=wasip1 GOARCH=wasm llgo build -o $BUILD_TEMP_DIR/buildcache.wasm -tags=nogc -debug ." \
        "$BUILD_TEMP_DIR/buildcache.wasm" \
        "$LLGO_IWASM --stack-size=819200000 --heap-size=800000000 $BUILD_TEMP_DIR/buildcache.wasm"
else
    echo ""
    echo -e "${BLUE}Skipping WASM tests (iwasm not available)${NC}"
fi

# ===========================================================
# ESP32-C3 Embedded Build Cache Tests
# ===========================================================

# ESP32-C3 tests use the same buildcache project, just with different target
echo ""
echo -e "${BLUE}Running ESP32-C3 tests...${NC}"
run_test_suite "esp32c3" \
    "llgo build -target=esp32c3 -o $BUILD_TEMP_DIR/buildcache.elf -debug ." \
    "$BUILD_TEMP_DIR/buildcache.elf"

# ===========================================================
# Summary
# ===========================================================
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}Failed: 0${NC}"
    echo -e "${GREEN}All cache tests passed!${NC}"
fi
