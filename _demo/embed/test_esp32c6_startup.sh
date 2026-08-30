#!/usr/bin/env bash
# ESP32-C6 target, startup, ISA, and firmware-image regression test.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMP_DIR=$(mktemp -d "$SCRIPT_DIR/.test_esp32c6_tmp.XXXXXX")
TEST_ELF="$TEMP_DIR/test.elf"
TEST_BIN="$TEMP_DIR/test.bin"

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

for tool in llgo llvm-nm llvm-objdump llvm-readelf; do
    if ! command -v "$tool" > /dev/null 2>&1; then
        echo "FAIL: required tool not found: $tool"
        exit 1
    fi
done

ESPTOOL_PYTHON=()
for python_cmd in python3 python; do
    if command -v "$python_cmd" > /dev/null 2>&1 && "$python_cmd" -c 'import esptool' &> /dev/null; then
        ESPTOOL_PYTHON=("$python_cmd")
        break
    fi
done
if [ ${#ESPTOOL_PYTHON[@]} -eq 0 ] && command -v py > /dev/null 2>&1 && py -3 -c 'import esptool' &> /dev/null; then
    ESPTOOL_PYTHON=(py -3)
fi
if [ ${#ESPTOOL_PYTHON[@]} -eq 0 ]; then
    echo "FAIL: esptool not found; install esptool==5.1.0"
    exit 1
fi

echo "==> Building ESP32-C6 ELF and ROM image..."
pushd "$SCRIPT_DIR" > /dev/null
llgo build -a -target=esp32c6 -o "$TEMP_DIR/test" -obin ./esp32c6/atomic
popd > /dev/null

if [ ! -s "$TEST_ELF" ] || [ ! -s "$TEST_BIN" ]; then
    echo "FAIL: ESP32-C6 build did not produce both ELF and BIN outputs"
    exit 1
fi

echo "==> Verifying ESP32-C6 ISA..."
ARCH=$(llvm-readelf -A "$TEST_ELF" | awk '/Value: rv32/ { print $2; exit }')
case "$ARCH" in
    rv32*_m*_a*_c*_zicsr*_zifencei*) ;;
    *)
        echo "FAIL: unexpected RISC-V architecture attributes: $ARCH"
        exit 1
        ;;
esac
if ! llvm-objdump -d "$TEST_ELF" | grep -E '[[:space:]]amo(add|swap|and|or|xor|min|max)\.w(\.[a-z]+)?[[:space:]]' > /dev/null; then
    echo "FAIL: sync/atomic did not lower to an RV32A AMO instruction"
    exit 1
fi

echo "==> Verifying startup and USB Serial/JTAG support..."
if ! llvm-objdump -d "$TEST_ELF" | sed -n '/<_start>:/,+50p' | grep '__libc_init_array' > /dev/null; then
    echo "FAIL: _start does not call __libc_init_array"
    exit 1
fi
for symbol in _write USB_SERIAL_JTAG; do
    if ! llvm-nm "$TEST_ELF" | awk -v symbol="$symbol" '$NF == symbol { found = 1 } END { exit !found }'; then
        echo "FAIL: required symbol not found: $symbol"
        exit 1
    fi
done
USB_SERIAL_JTAG_ADDR=$(llvm-nm "$TEST_ELF" | awk '$NF == "USB_SERIAL_JTAG" { print $1; exit }')
if [ "$USB_SERIAL_JTAG_ADDR" != "6000f000" ]; then
    echo "FAIL: ESP32-C6 USB Serial/JTAG base is $USB_SERIAL_JTAG_ADDR, want 6000f000"
    exit 1
fi

echo "==> Verifying ESP32-C6 ROM image..."
if ! BIN_INFO=$("${ESPTOOL_PYTHON[@]}" -m esptool --chip esp32c6 image-info "$TEST_BIN" 2>&1); then
    echo "$BIN_INFO"
    echo "FAIL: esptool could not parse the ESP32-C6 image"
    exit 1
fi
for expected in 'ESP32C6 Image Header' 'Chip ID: 13 \(ESP32-C6\)' 'Checksum: .* \(valid\)' 'Validation hash: .* \(valid\)'; do
    if ! grep -E "$expected" <<< "$BIN_INFO" > /dev/null; then
        echo "$BIN_INFO"
        echo "FAIL: image metadata missing: $expected"
        exit 1
    fi
done
if grep 'Invalid flash frequency' <<< "$BIN_INFO" > /dev/null; then
    echo "$BIN_INFO"
    echo "FAIL: ESP32-C6 image uses an invalid flash-frequency code"
    exit 1
fi

if command -v esp-emu > /dev/null 2>&1; then
    echo "==> Running ESP32-C6 UART target in esp-emu..."
    pushd "$SCRIPT_DIR" > /dev/null
    if ! EMULATOR_OUTPUT=$(llgo run -a -target=esp32c6-emulator -emulator ./testdata/esp32-serial/chello 2>&1); then
        popd > /dev/null
        echo "$EMULATOR_OUTPUT"
        echo "FAIL: LLGo could not execute the ESP32-C6 image in esp-emu"
        exit 1
    fi
    popd > /dev/null
    if ! grep 'Hello World' <<< "$EMULATOR_OUTPUT" > /dev/null; then
        echo "$EMULATOR_OUTPUT"
        echo "FAIL: ESP32-C6 emulator output mismatch"
        exit 1
    fi
else
    echo "SKIP: esp-emu is unavailable; build/image validation still passed"
fi

echo "PASS: ESP32-C6 target, RV32IMAC ISA, startup, USB serial, ROM image, and available emulator coverage"
