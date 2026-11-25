#!/usr/bin/env bash
set -euo pipefail

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
package=./test/std/sync/atomic
symbols=_github.com/xgo-dev/llgo/test/std/sync/atomic_test
test_cases='^Test(AddUint64|CompareAndSwapUint64)$'

check_instructions() {
    local symbol=$1
    shift
    local assembly
    assembly=$(llvm-objdump --disassemble-symbols="$symbols.$symbol" "$work_dir/atomic.test")
    for instruction in "$@"; do
        if ! grep -Eq "[[:space:]]${instruction}[[:space:]]" <<< "$assembly"; then
            printf 'Expected %s in %s\n%s\n' "$instruction" "$symbol" "$assembly" >&2
            return 1
        fi
    done
}

for target in 'v8.0 full' 'v8.0,lse full' 'v8.0,lse thin'; do
    read -r isa mode <<< "$target"
    echo "Checking GOARM64=$isa with $mode LTO"
    GOOS=darwin GOARCH=arm64 GOARM64="$isa" llgo test -c -O3 "-lto=$mode" \
        -o "$work_dir/atomic.test" "$package"
    "$work_dir/atomic.test" "-test.run=$test_cases" -test.v

    if [[ "$isa" == 'v8.0' ]]; then
        check_instructions addUint64 ldaxr stlxr
        check_instructions compareAndSwapUint64 ldaxr stlxr
    else
        check_instructions addUint64 ldaddal
        check_instructions compareAndSwapUint64 casal
    fi
done
