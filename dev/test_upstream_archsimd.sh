#!/usr/bin/env bash

# Run the qualified upstream instruction tests without copying generated tests
# or their internal helpers out of GOROOT. See dev/simd/README.md.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
: "${LLGO:=llgo}"
export GOEXPERIMENT=simd
arch=$(go env GOARCH)
case "$arch" in
  amd64)
    # This profile qualifies 128/256-bit operations. Upstream mixed-width tests
    # also contain 512-bit branches, which LLGo does not implement yet.
    export GOAMD64=v1
    export GODEBUG="${GODEBUG:+$GODEBUG,}cpu.avx512f=off"
    ;;
  arm64) ;;
  *) echo "Upstream archsimd: no qualified profile for $arch"; exit 0 ;;
esac

if [[ -n "${LLGO_SIMD_ARTIFACT_DIR:-}" ]]; then
  output=$LLGO_SIMD_ARTIFACT_DIR
  mkdir -p "$output"
else
  output=$(mktemp -d)
  trap 'rm -rf "$output"' EXIT
fi

expected="$output/upstream-archsimd-$arch-expected.txt"
: > "$expected"
filter='^('
separator=''
while IFS= read -r name; do
  # Git may check out the case lists with CRLF on Windows.
  name=${name%$'\r'}
  case "$name" in ''|'#'*) continue ;; esac
  [[ "$name" =~ ^Test[A-Za-z0-9_]+$ ]] || { echo "Invalid upstream case: $name" >&2; exit 1; }
  printf '%s\n' "$name" >> "$expected"
  filter+="$separator$name"
  separator='|'
done < "dev/simd/upstream-archsimd-$arch.txt"
[[ -s "$expected" ]] || { echo "Empty upstream case list" >&2; exit 1; }
filter+=')$'
package=simd/archsimd/internal/simd_test

echo "Upstream archsimd: $arch official Go reference"
go test -v -count=1 -timeout=5m -run="$filter" "$package"
binary="$output/upstream-archsimd-$arch.exe"
"$LLGO" test -O2 -c -o "$binary" "$package"
actual="$output/upstream-archsimd-$arch-actual.txt"
# testing writes names to stdout; runtime diagnostics on stderr must remain
# visible in the CI log without becoming entries in the case list.
list_status=0
"$binary" -test.list="$filter" > "$actual.raw" 2> "$actual.stderr.raw" || list_status=$?
cat "$actual.stderr.raw" >&2
[[ "$list_status" == 0 ]] || exit "$list_status"
# Native Windows executables may also emit CRLF. Normalize only the line ending
# before checking the exact set of test names; retain the raw output for diagnosis.
sed $'s/\r$//' "$actual.raw" > "$actual"
sed $'s/\r$//' "$actual.stderr.raw" > "$actual.stderr"
if [[ ! -s "$actual" ]] && grep -q '^Skipping tests: AVX is not available$' "$actual.stderr"; then
  exit 0
fi
# A renamed/removed upstream case must not turn the gate into a silent no-op.
sort "$expected" > "$expected.sorted"
sort "$actual" > "$actual.sorted"
diff -u "$expected.sorted" "$actual.sorted"
echo "Upstream archsimd: $arch LLGo O2"
"$binary" -test.v -test.count=1 -test.timeout=5m -test.run="$filter"
