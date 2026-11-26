#!/usr/bin/env bash
# Run on a Linux/amd64 LLGo build host with Intel SDE available.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
: "${SDE:=sde64}" "${LLGO:=llgo}"
export GOEXPERIMENT=simd GOAMD64=v1
output=$(mktemp -d)
trap 'rm -rf "$output"' EXIT
for compiler in go "$LLGO"; do
  build=("$compiler" test)
  if [[ "$compiler" != go ]]; then build+=(-O2); fi
  for package in ./test/simd ./test/simd/portable ./test/std/simd simd/archsimd/internal/simd_test; do
    "${build[@]}" -c -o "$output/test" "$package"
    filter=.
    if [[ "$package" == ./test/simd ]]; then filter='^TestSIMD(512|X86)'; fi
    GODEBUG="${GODEBUG:+$GODEBUG,}simd=512" "$SDE" -spr -- "$output/test" -test.count=1 -test.timeout="${SDE_TIMEOUT:-60m}" -test.v -test.run="$filter" >"$output/result" || { cat "$output/result"; exit 1; }
    cat "$output/result"
    if [[ "$package" == ./test/simd ]]; then
      # Sapphire Rapids supplies YMM and ZMM state even with effective Go queries off.
      GODEBUG=cpu.all=off LLGO_TEST_MASKED_YMM=1 LLGO_TEST_MASKED_ZMM=1 "$SDE" -spr -- "$output/test" -test.count=1 \
        -test.run='^TestSIMD(FMVOutlineCallers|512FixedRegisterCall)$' -test.v
      GODEBUG="${GODEBUG:+$GODEBUG,}cpu.avx512f=off,cpu.sha=off" "$SDE" -spr -- "$output/test" -test.count=1 -test.run='^TestSIMD512(VBMI2|EitherGuard|IndependentGuards)WideCall$' -test.v | tee "$output/result"
      grep -q 'AVX2=true AVX512=false VBMI2=true' "$output/result"
      grep -q 'FMA=true SHA=false AVXVNNI=true VBMI2=true' "$output/result"
    elif [[ "$package" == ./test/simd/portable ]]; then
      grep -q 'width=512 emulated=false AVX512=true' "$output/result"
    elif [[ "$package" == simd/archsimd/internal/simd_test ]]; then
      awk '/^--- PASS: Test/ {print $3}' "$output/result" | sort >"$output/passed"
      awk '/^Test/ {print $1}' dev/simd/upstream-archsimd-amd64.txt | sort >"$output/expected"
      diff -u "$output/expected" "$output/passed"
    fi
  done
done

# Exercise Clang's textual IR path with a v3 callee and an AVX512 caller.
GOAMD64=v3 "$LLGO" test -O2 -gen-llfiles -lto=thin -c -o "$output/test" ./test/simd
"$SDE" -spr -- "$output/test" -test.count=1 -test.run='^TestSIMD256CrossPackageAVX512$' -test.v | tee "$output/result"
grep -q '^--- PASS: TestSIMD256CrossPackageAVX512' "$output/result"
