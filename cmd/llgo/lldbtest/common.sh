#!/bin/bash

LLDB_TEST_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Function to find LLDB 18+
find_lldb() {
    local lldb_paths=()
    if [ -n "${LLGO_LLDB:-}" ]; then
        lldb_paths+=("$LLGO_LLDB")
    fi
    lldb_paths+=(
        "/opt/homebrew/bin/lldb"
        "/usr/local/bin/lldb"
        "/usr/bin/lldb"
        "lldb"  # This will use the system PATH
    )

    for lldb_path in "${lldb_paths[@]}"; do
        if command -v "$lldb_path" >/dev/null 2>&1; then
            local version
            version=$("$lldb_path" --version | grep -oE '[0-9]+' | head -1)
            if [[ "$version" =~ ^[0-9]+$ ]] && [ "$version" -ge 18 ]; then
                echo "$lldb_path"
                return 0
            fi
        fi
    done

    echo "Error: LLDB 18 or higher not found" >&2
    exit 1
}

# Find LLDB 18+
LLDB_PATH=$(find_lldb)
echo "LLDB_PATH: $LLDB_PATH"
"$LLDB_PATH" --version
export LLDB_PATH

# Default package path
export DEFAULT_PACKAGE_PATH="$LLDB_TEST_ROOT"
export LLDB_TEST_OPTLEVEL="${LLDB_TEST_OPTLEVEL:--O0}"

# Function to build the project
build_project() {
    # package_path parameter is kept for backward compatibility
    local current_dir
    current_dir=$(pwd) || return

    if ! cd "${DEFAULT_PACKAGE_PATH}"; then
        echo "Failed to change directory to ${DEFAULT_PACKAGE_PATH}" >&2
        return 1
    fi

    llgo build "${LLDB_TEST_OPTLEVEL}" -ldflags=-w=false -o "debug.out" . || {
        local ret=$?
        cd "$current_dir" || return
        return $ret
    }

    cd "$current_dir" || return
}
