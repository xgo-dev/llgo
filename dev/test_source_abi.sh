#!/usr/bin/env bash
# Exact-version, real application regressions for explicit ABI adaptation.
set -euo pipefail

repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo"
export GOWORK=off GOFLAGS= CGO_ENABLED=1
llgo=$(command -v "${LLGO:-llgo}")
llgo=$(realpath "$llgo")
export LLGO_ROOT="$repo"

[[ $(go env GOOS)/$(go env GOARCH) == linux/amd64 ]] || {
  echo 'source ABI regressions require linux/amd64' >&2
  exit 1
}
[[ $(go env GOVERSION) == go1.27.* ]] || {
  echo 'source ABI regressions require Go 1.27' >&2
  exit 1
}
[[ $("$llgo" env LLGO_LLVM_VERSION) == 22.* ]] || {
  echo 'source ABI regressions require LLVM 22' >&2
  exit 1
}

mkdir -p "$repo/_out"
report=$(mktemp -d "$repo/_out/source-abi.XXXXXX")
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
sha256sum "$llgo" > "$report/compiler.sha256"
go version -m "$llgo" > "$report/compiler-build.txt"
go version > "$report/go-version.txt"
"$llgo" env -json LLGO_LLVM_CONFIG LLGO_LLVM_VERSION LLGO_CLANG LLGO_CLANG_VERSION LLGO_CLANGXX > "$report/llvm-environment.json"
clang=$("$llgo" env LLGO_CLANG)
"$clang" --version > "$report/clang-version.txt"
grep -Eq 'clang version 22\.' "$report/clang-version.txt" || {
  echo 'source ABI regressions require the selected Clang 22' >&2
  exit 1
}
go build -o "$work/adapter" "$repo/chore/ccgobridge"

cases=("$@")
if [[ ${#cases[@]} == 0 ]]; then
  cases=(sqlite purego turso libccallback crc32)
fi
failed=0
for name in "${cases[@]}"; do
  dir="$repo/chore/ccgobridge/testdata/$name"
  case "$name" in
    crc32)
      args=()
      ;;
    sqlite)
      args=(modernc.org/libc modernc.org/sqlite/lib modernc.org/sqlite)
      ;;
    libccallback)
      args=(modernc.org/libc)
      ;;
    purego|turso)
      args=(-profile purego-v0.9.1 -runtime "$repo/runtime" github.com/ebitengine/purego)
      ;;
    *) echo "unknown case: $name" >&2; exit 1 ;;
  esac
  stage="$work/stage-$name"
  if [[ ${#args[@]} != 0 ]]; then
    if ! timeout -k 10s 10m "$work/adapter" -dir "$dir" -out "$stage" "${args[@]}" > "$report/$name.adapt.log" 2>&1; then
      printf '{"case":"%s","phase":"adapt","status":"failed"}\n' "$name" >> "$report/results.jsonl"
      failed=1
      continue
    fi
    cp "$stage/adaptations.json" "$report/$name.adaptations.json"
    cp "$stage/environment.json" "$report/$name.environment.json"
  fi
  compilers=(go llgo)
  if [[ $name == sqlite || $name == libccallback ]]; then
    compilers=(go go-adapted llgo)
  fi
  for compiler in "${compilers[@]}"; do
    tool=go
    flags=(-mod=readonly)
    if [[ $compiler != go && ${#args[@]} != 0 ]]; then
      flags+=(-modfile="$stage/build.mod")
    fi
    if [[ $compiler == llgo ]]; then
      tool="$llgo"
    fi
    binary="$work/$name-$compiler"
    status=passed
    phase=build
    if (cd "$dir" && timeout -k 10s 20m "$tool" build "${flags[@]}" -o "$binary" .) > "$report/$name-$compiler.build.log" 2>&1; then
      phase=run
      sha256sum "$binary" > "$report/$name-$compiler.sha256"
      if ! timeout -k 10s 2m "$binary" > "$report/$name-$compiler.run.log" 2>&1; then
        status=failed
      fi
      rm -- "$binary"
    else
      status=failed
    fi
    printf '{"case":"%s","compiler":"%s","phase":"%s","status":"%s"}\n' \
      "$name" "$compiler" "$phase" "$status" >> "$report/results.jsonl"
    echo "$name $compiler: $status ($phase)"
    if [[ $status == failed ]]; then
      failed=1
    fi
  done
done
echo "Reports: $report"
exit "$failed"
