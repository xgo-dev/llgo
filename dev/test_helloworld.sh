#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 || ! "$1" =~ ^1\.[0-9]+$ ]]; then
	echo "usage: $0 <go.mod version>" >&2
	exit 2
fi
mod_version=$1

llgo_cmd="${LLGO:-llgo}"
if [[ "${llgo_cmd}" != */* ]]; then
	llgo_cmd="$(command -v "${llgo_cmd}")"
elif [[ "${llgo_cmd}" != /* ]]; then
	llgo_cmd="$(cd "$(dirname "${llgo_cmd}")" && pwd)/$(basename "${llgo_cmd}")"
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-hello.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
export GOTOOLCHAIN=local

hello_dir="${work_dir}/helloworld"
mkdir -p "${hello_dir}"
cat >"${hello_dir}/go.mod" <<EOF
module hello

go ${mod_version}

require github.com/goplus/lib v0.3.1
EOF
cat >"${hello_dir}/main.go" <<'EOF'
package main

import (
	"fmt"

	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/std"
)

func main() {
	fmt.Println("Hello, LLGo!")
	println("Hello, LLGo!")
	c.Printf(c.Str("Hello, LLGo!\n"))
	c.Printf(std.Str("Hello LLGo by cpp/std.Str\n").CStr())
}
EOF
(
	cd "${hello_dir}"
	go mod tidy
	if output="$("${llgo_cmd}" run . 2>&1)"; then
		:
	else
		status=$?
		printf '%s\n' "${output}" >&2
		exit "${status}"
	fi
	printf '%s\n' "${output}"
	hello_count="$(grep -Fxc "Hello, LLGo!" <<<"${output}" || true)"
	if [[ "${hello_count}" -ne 3 ]]; then
		echo "expected three Hello, LLGo! lines, got ${hello_count}" >&2
		exit 1
	fi
	if ! grep -Fqx "Hello LLGo by cpp/std.Str" <<<"${output}"; then
		echo "missing cpp/std.Str output" >&2
		exit 1
	fi
)

if [[ "${LLGO_HELLO_EMBED:-1}" != 0 && "${LLGO_HELLO_EMBED:-1}" != false ]]; then
	embed_dir="${work_dir}/embed"
	mkdir -p "${embed_dir}"
	cat >"${embed_dir}/go.mod" <<EOF
module embed-smoke

go ${mod_version}
EOF
	cat >"${embed_dir}/main.go" <<'EOF'
package main

func main() {}
EOF
	(
		cd "${embed_dir}"
		"${llgo_cmd}" build -v -target esp32-coreboard-v2 -o demo.out .
		test -f demo.out.elf
	)
fi

echo "Hello World smoke test passed with $(go env GOVERSION), go.mod ${mod_version}"
