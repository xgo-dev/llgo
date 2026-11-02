#!/bin/bash

set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)

# shellcheck source=../runtime/common.sh
# shellcheck disable=SC1091
source "$script_dir/../runtime/common.sh"

LLGO=${LLGO:-llgo}

test_tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/llgo-native-debug.XXXXXX")
trap 'rm -rf "$test_tmp_dir"' EXIT
artifact="$test_tmp_dir/native-debug.out"
optimized_artifact="$test_tmp_dir/native-debug-o2.out"

cd "$script_dir"
"$LLGO" build -O0 -o "$artifact" .
"$LLGO" build -O2 -o "$optimized_artifact" .

export LLGO_NATIVE_DEBUG_SOURCE="$script_dir"
export LLGO_NATIVE_DEBUG_ARTIFACT="$artifact"
export LLGO_NATIVE_DEBUG_OPTIMIZED_ARTIFACT="$optimized_artifact"
lldb_log="$test_tmp_dir/lldb.log"
"$LLDB_PATH" --batch \
    -O 'settings set symbols.enable-external-lookup false' "$artifact" \
    -o 'script import os, runpy; _ = runpy.run_path(os.path.join(os.environ["LLGO_NATIVE_DEBUG_SOURCE"], "acceptance.py")); _["main"]()' 2>&1 | tee "$lldb_log"

if grep -Fq 'Traceback (most recent call last)' "$lldb_log" || \
    ! grep -Fq 'NATIVE_DEBUG_ACCEPTANCE_OK' "$lldb_log"; then
    exit 1
fi
