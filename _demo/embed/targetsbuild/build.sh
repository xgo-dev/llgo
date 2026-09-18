#!/bin/bash

# Function to display usage information
show_usage() {
	cat << EOF
Usage: $(basename "$0") [OPTIONS] <TEST_DIR> [TARGET_FILE]

Build targets for llgo across multiple platforms.

OPTIONS:
    -h, --help      Show this help message and exit

ARGUMENTS:
    TEST_DIR        Required. The test directory containing main.go to build.
                    Examples: empty, defer

    TARGET_FILE     Optional. A text file containing target names, one per line.
                    Lines starting with # are treated as comments and ignored.
                    Empty lines are also ignored.

BEHAVIOR:
    Without TARGET_FILE:
        - Automatically discovers all targets from ../../targets/*.json files
        - Extracts target names from JSON filenames

    With TARGET_FILE:
        - Reads target names from the specified file
        - Supports comments (lines starting with #)
        - Ignores empty lines and whitespace

IGNORED TARGETS:
    The following targets are automatically ignored and not built:
    atmega1280, atmega2560, atmega328p, atmega32u4, attiny85,
    fe310, k210, riscv32, riscv64, rp2040

RESULT CATEGORIES:
    ✅ Successful: Build completed successfully
    🔕 Ignored: Target is in the ignore list
    ⚠️  Warned: Build failed with configuration warnings
    ❌ Failed: Build failed with errors

EXIT CODES:
    0    All builds successful, ignored, or warned only
    1    One or more builds failed with errors

EXAMPLES:
    $(basename "$0") empty                    # Build empty test for all targets
    $(basename "$0") defer                    # Build defer test for all targets
    $(basename "$0") empty my-targets.txt    # Build empty test for specific targets
    $(basename "$0") --help                   # Show this help

TARGET FILE FORMAT:
    # This is a comment
    esp32
    cortex-m4

    # Another comment
    riscv64
EOF
}

# Check for help flag
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
	show_usage
	exit 0
fi

# Check for required TEST_DIR argument
if [ $# -lt 1 ]; then
	echo "Error: TEST_DIR is required."
	echo "Use '$(basename "$0") --help' for usage information."
	exit 1
fi

# Check for invalid number of arguments
if [ $# -gt 2 ]; then
	echo "Error: Too many arguments."
	echo "Use '$(basename "$0") --help' for usage information."
	exit 1
fi

# Get test directory
test_dir="$1"
if [ ! -d "$test_dir" ]; then
	echo "Error: Test directory '$test_dir' not found."
	echo "Use '$(basename "$0") --help' for usage information."
	exit 1
fi

echo "Testing: $test_dir"
echo "----------------------------------------"

# Initialize arrays to store results
successful_targets=()
ignored_targets=()
warned_targets=()
failed_targets=()
targets_to_build=()
delegated_targets=()

# Define ignore list based on test directory
case "$test_dir" in
	empty)
		ignore_list=(
			"atmega1280"
			"atmega2560"
			"atmega328p"
			"atmega32u4"
			"attiny85"
			"fe310"
			"k210"
			"riscv32"
			"riscv64"
			"rp2040"
		)
		;;
	defer)
		ignore_list=(
			"atmega1280"
			"atmega2560"
			"atmega328p"
			"atmega32u4"
			"attiny85"
			"fe310"
			"k210"
			"riscv32"
			"riscv64"
			"rp2040"

			# <unknown>:0: error: out of range branch target (expected an integer in the range -4096 to 4095)
			# error: cannot compile inline asm
			"digispark"

			#In file included from /home/runner/work/llgo/llgo/runtime/internal/clite/debug/_wrap/debug.c:9:
			# In file included from /usr/include/dlfcn.h:22:
			# In file included from /usr/include/features.h:394:
			# /usr/include/features-time64.h:20:10: fatal error: 'bits/wordsize.h' file not found
			#    20 | #include <bits/wordsize.h>
			#       |          ^~~~~~~~~~~~~~~~~
			# 1 error generated.
			# panic: exit status 1
			"nintendoswitch"

			# ld.lld: error: /home/runner/.cache/llgo/crosscompile/picolibc-v0.1.0/libc-avr.a(-home-runner-.cache-llgo-crosscompile-picolibc-v0.1.0-newlib-libc-tinystdio-printf.c1170247836.o): cannot link object files with incompatible target ISA
			"simavr"

			# libc symbol lack
			"arduino-leonardo"
			"arduino-mega1280"
			"arduino-mega2560"
			"arduino-nano-new"
			"arduino-nano"
			"arduino"
			"atmega1284p"
			"atmega328pb"
			"attiny1616"
			"cortex-m0"
			"cortex-m0plus"
			"cortex-m3"
			"cortex-m33"
			"cortex-m4"
			"cortex-m7"
			"d1mini"
			"esp8266"
			"gameboy-advance"
			"hifive1b"
			"maixbit"
			"nodemcu"
			"riscv-qemu"
			"riscv32-esp"
			"stm32l0x2"
			"tkey"
		)
		;;
	*)
		echo "Error: Unknown test directory '$test_dir'. Please add ignore_list in build.sh."
		exit 1
		;;
esac

# Build the targets list based on input method
if [ $# -eq 2 ]; then
	# Read targets from file
	target_file="$2"
	if [ ! -f "$target_file" ]; then
		echo "Error: Target file '$target_file' not found."
		echo "Use '$(basename "$0") --help' for usage information."
		exit 1
	fi

	while IFS= read -r target || [[ -n "$target" ]]; do
		# Skip empty lines and comments
		if [[ -z "$target" || "$target" =~ ^[[:space:]]*# ]]; then
			continue
		fi

		# Trim whitespace
		target=$(echo "$target" | xargs)
		targets_to_build+=("$target")
	done < "$target_file"
else
	# Use targets from *.json files
	for target_file in ../../../targets/*.json; do
		# Extract target name from filename (remove path and .json extension)
		target=$(basename "$target_file" .json)
		targets_to_build+=("$target")
	done
fi

# Process each target
for target in "${targets_to_build[@]}"; do
	# WebAssembly ecosystem profiles need their own linker, output form, and
	# runtime probe. The wasm-runtime CI job exercises them instead of treating
	# JS and WASI modules as a generic hello.elf embedded target.
	case "$target" in
		emscripten|emscripten-memory64|wasi|wasip1|wasm)
			echo 🔀 $target "(covered by WebAssembly profile tests)"
			delegated_targets+=("$target")
			continue
			;;
	esac

	# Check if target is in ignore list
	if [[ " ${ignore_list[@]} " =~ " ${target} " ]]; then
		echo 🔕 $target "(ignored)"
		ignored_targets+=("$target")
		continue
	fi

	output=$(../../../dev/llgo.sh build -target $target -o hello.elf "./$test_dir" 2>&1)
	if [ $? -eq 0 ]; then
		echo ✅ $target `file hello.elf`
		successful_targets+=("$target")
	else
		# Check if output contains warning messages
		if echo "$output" | grep -q "does not have a valid LLVM target triple\|does not have a valid CPU configuration"; then
			echo ⚠️ $target
			echo "$output"
			warned_targets+=("$target")
		else
			echo ❌ $target
			echo "$output"
			failed_targets+=("$target")
		fi
	fi
done

echo ""
echo "----------------------------------------"

# Output successful targets
echo "Successful targets (${#successful_targets[@]} total):"
for target in "${successful_targets[@]}"; do
	echo "$target"
done

echo ""
echo "WebAssembly profile targets (${#delegated_targets[@]} total):"
for target in "${delegated_targets[@]}"; do
	echo "$target"
done

echo ""
echo "Ignored targets (${#ignored_targets[@]} total):"
for target in "${ignored_targets[@]}"; do
	echo "$target"
done

echo ""
echo "Warned targets (${#warned_targets[@]} total):"
for target in "${warned_targets[@]}"; do
	echo "$target"
done

echo ""
echo "Failed targets (${#failed_targets[@]} total):"
for target in "${failed_targets[@]}"; do
	echo "$target"
done

# Exit with error code if there are any failed targets
if [ ${#failed_targets[@]} -gt 0 ]; then
	echo ""
	echo "Build failed with ${#failed_targets[@]} failed targets."
	exit 1
fi
