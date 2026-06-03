#!/bin/bash
# ESP32-C3 startup/linker regression test
#
# Keep default ESP smoke coverage in test-esp-serial-startup.sh.
# This script is dedicated to ESP32-C3-specific regressions.
#
# Verifies:
# 1. _start uses newlib's __libc_init_array (not TinyGo's start.S)
# 2. __init_array_start symbol is present in ELF
# 3. __init_array_start is included in BIN load segments
# 4. float output regression case(s) remain stable on esp32c3-basic emulator

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Create temp dir inside _demo/embed/ to use existing go.mod
TEMP_DIR="$SCRIPT_DIR/.test_tmp_$$"
mkdir -p "$TEMP_DIR"
TEST_GO="$TEMP_DIR/main.go"
TEST_ELF="test.elf"
TEST_BIN="test.bin"

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

extract_last_nonempty_lines() {
    local n="$1"
    awk -v n="$n" '
    NF { out[++count] = $0 }
    END {
        if (n <= 0 || count == 0) {
            exit
        }
        start = count - n + 1
        if (start < 1) {
            start = 1
        }
        for (i = start; i <= count; i++) {
            print out[i]
        }
    }'
}

run_case_and_compare() {
    local case_dir="$1"
    local expected="$2"
    local raw_output
    local actual
    local expected_lines

    echo "Running: llgo run -a -target=esp32c3-basic -emulator $case_dir"
    if ! raw_output=$(llgo run -a -target=esp32c3-basic -emulator "$case_dir" 2>&1); then
        echo "✗ FAIL: command failed for $case_dir"
        echo "$raw_output"
        return 1
    fi

    expected_lines=$(printf "%s\n" "$expected" | awk 'NF { n++ } END { print n + 0 }')
    actual=$(printf "%s\n" "$raw_output" | tr -d '\r' | extract_last_nonempty_lines "$expected_lines")
    if [ "$actual" = "$expected" ]; then
        echo "✓ PASS: $case_dir"
        return 0
    fi

    echo "✗ FAIL: output mismatch for $case_dir"
    echo "Expected:"
    printf "%s\n" "$expected"
    echo ""
    echo "Got:"
    printf "%s\n" "$actual"
    echo ""
    echo "Diff:"
    diff -u <(printf "%s\n" "$expected") <(printf "%s\n" "$actual") || true
    return 1
}

# Check if esptool.py is installed
# esptool.py is required to parse ESP32-C3 BIN file format and verify
# that constructor-related data is included in the firmware
if ! command -v esptool.py &> /dev/null; then
    echo "✗ FAIL: esptool.py not found"
    echo "Please install: pip3 install esptool==5.1.0"
    exit 1
fi

echo "==> Creating minimal test program..."
cat > "$TEST_GO" << 'EOF'
package main

import "github.com/goplus/lib/c"

func main() {
    println("Hello World\n")
	c.Printf(c.Str("Hello World\n"))
}
EOF

echo "==> Building for ESP32-C3 target (ELF + BIN)..."
cd "$TEMP_DIR"
llgo build -target=esp32c3 -o test -obin "$TEST_GO"

if [ ! -f "$TEST_ELF" ]; then
    echo "✗ FAIL: Build failed, $TEST_ELF not found"
    exit 1
fi

if [ ! -f "$TEST_BIN" ]; then
    echo "✗ FAIL: BIN file not generated, $TEST_BIN not found"
    exit 1
fi

echo ""
echo "=== Test 1: Verify newlib startup (calls __libc_init_array) ==="

# Disassemble _start and check for __libc_init_array call
if llvm-objdump -d "$TEST_ELF" | grep -A50 "<_start>:" | grep "__libc_init_array" > /dev/null; then
    echo "✓ PASS: _start calls __libc_init_array"
    echo "        ESP32-C3 uses newlib's standard startup"
else
    echo "✗ FAIL: _start does NOT call __libc_init_array"
    echo "        ESP32-C3 should use newlib's startup flow"
    echo ""
    echo "Expected inheritance: esp32c3 → riscv32-nostart → riscv-nostart → riscv-basic"
    echo "Current _start disassembly:"
    llvm-objdump -d "$TEST_ELF" | grep -A50 "<_start>:" || true
    exit 1
fi

echo ""
echo "=== Test 2: Verify __init_array_start symbol (ELF) ==="

# Get __init_array_start symbol address
# This symbol marks where .init_array is placed by the linker script
#
# Real output from: llvm-nm "$TEST_ELF"
#
# 40380450 d __init_array_start
# ^$1      ^$2 ^$3
# (Addr)   (Type: d=local data symbol in .rodata)
#
# Field breakdown:
#   $1 = 40380450          (symbol address in hex without 0x prefix)
#   $2 = d                 (symbol type: d=local data, D=global data, T=text, etc.)
#   $3 = __init_array_start (symbol name)
#
# We extract $1 and add "0x" prefix: 40380450 → 0x40380450
INIT_ARRAY_START=$(llvm-nm "$TEST_ELF" | grep "__init_array_start" | awk '{print "0x"$1}')
if [ -z "$INIT_ARRAY_START" ]; then
    echo "✗ FAIL: __init_array_start symbol not found"
    exit 1
fi

echo "✓ PASS: __init_array_start found at $INIT_ARRAY_START"
INIT_ADDR_DEC=$((INIT_ARRAY_START))

INIT_ARRAY_INFO=$(llvm-readelf -S "$TEST_ELF" | grep "\.init_array")
if [ -z "$INIT_ARRAY_INFO" ]; then
    echo "✗ FAIL: .init_array section not found"
    exit 1
fi
INIT_ARRAY_SIZE=$(echo "$INIT_ARRAY_INFO" | awk '{print "0x"$7}')
INIT_ARRAY_SIZE_DEC=$((INIT_ARRAY_SIZE))
echo ".init_array section size: $INIT_ARRAY_SIZE"

echo ""
echo "=== Test 3: Verify __init_array_start included in BIN file ==="

# Get BIN file segment information using esptool.py
# ESP32-C3 BIN files contain multiple segments with load addresses.
# We need to verify the __init_array_start address is covered by one segment.
#
# Real output from: esptool.py --chip esp32c3 image_info test.bin
#
# Segments Information
# ====================
# Segment   Length   Load addr   File offs  Memory types
# -------  -------  ----------  ----------  ------------
#       0  0x000f0  0x3fc84468  0x00000018  DRAM
#       1  0x00004  0x3fc84558  0x00000110  DRAM
#       2  0x0006c  0x3fc8455c  0x0000011c  DRAM
#       3  0x0042c  0x40380000  0x00000190  IRAM
#       4  0x0003c  0x4038042c  0x000005c4  IRAM  ← This is .rodata!
#          ^$2      ^$3         ^$4
#          (Length) (LoadAddr)  (FileOffset)
#
# Field breakdown for segment line:
#   $1 = 4           (segment number)
#   $2 = 0x0003c     (segment length/size)
#   $3 = 0x4038042c  (load address - matches .rodata address from ELF!)
#   $4 = 0x000005c4  (file offset in BIN)
#   $5+ = IRAM       (memory type)
#
# We extract $2 (length) and $3 (load addr) to verify __init_array_start is within bounds
if ! BIN_INFO=$(esptool.py --chip esp32c3 image_info "$TEST_BIN" 2>&1); then
    echo "✗ FAIL: esptool.py failed to parse BIN file"
    echo "$BIN_INFO"
    exit 1
fi

FOUND_SEG=0
if [ $INIT_ARRAY_SIZE_DEC -eq 0 ]; then
    echo "✓ PASS: .init_array is empty; no constructor payload needs BIN segment coverage"
else
    INIT_ARRAY_END_DEC=$((INIT_ADDR_DEC + INIT_ARRAY_SIZE_DEC))
    while read -r SEG_NUM SEG_LEN SEG_LOAD; do
        SEG_LEN_DEC=$((SEG_LEN))
        SEG_LOAD_DEC=$((SEG_LOAD))
        SEG_END=$((SEG_LOAD_DEC + SEG_LEN_DEC))
        if [ $INIT_ADDR_DEC -ge $SEG_LOAD_DEC ] && [ $INIT_ARRAY_END_DEC -le $SEG_END ]; then
            OFFSET=$((INIT_ADDR_DEC - SEG_LOAD_DEC))
            echo "✓ PASS: .init_array is within BIN Segment $SEG_NUM (offset: +0x$(printf '%x' $OFFSET), size: $INIT_ARRAY_SIZE)"
            echo "        Segment range: [$SEG_LOAD, $(printf '0x%x' $SEG_END))"
            FOUND_SEG=1
            break
        fi
    done < <(echo "$BIN_INFO" | awk '$1 ~ /^[0-9]+$/ && $2 ~ /^0x/ && $3 ~ /^0x/ {print $1, $2, $3}')

    if [ $FOUND_SEG -eq 0 ]; then
        echo "✗ FAIL: .init_array payload is NOT within any BIN segment"
        echo "        .init_array range: [$INIT_ARRAY_START, $(printf '0x%x' $INIT_ARRAY_END_DEC))"
        echo ""
        echo "BIN segments:"
        echo "$BIN_INFO" | grep -A20 "Segments Information"
        exit 1
    fi
fi

echo ""
echo "=== Test 4: ESP32-C3 float output regressions (temporary) ==="
pushd "$SCRIPT_DIR" > /dev/null
FLOAT_1664_EXPECT=$'+5.000000e+00 +8.000000e+00\n1 +2.000000e+00\n0x0 +0.000000e+00 notOk: true\n0x0 +0.000000e+00 true\n3 +6.280000e+00'
if [[ "$(go env GOVERSION)" == go1.26* ]]; then
    FLOAT_1664_EXPECT=$'5 8\n1 2\n0x0 0 notOk: true\n0x0 0 true\n3 6.28'
fi
run_case_and_compare "./esp32c3/float-1664" "$FLOAT_1664_EXPECT"
# Mixed Go println + C printf(%f) regression tracking for issue #1723.
run_case_and_compare "./esp32c3/print-float-1723" $'go 0\nf=1.100000'
popd > /dev/null

echo ""
echo "=== All Tests Passed ==="
echo "✓ ESP32-C3 uses newlib startup (_start calls __libc_init_array)"
echo "✓ __init_array_start symbol exists in ELF"
echo "✓ .init_array payload is correctly handled in BIN (or empty)"
echo "✓ ESP32-C3 float output regression cases match expected output"
echo "✓ Constructor function pointers will be correctly flashed to ESP32-C3"

exit 0
