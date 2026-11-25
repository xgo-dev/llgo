#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'EOF'
usage: dev/test_std_pkg.sh <pkg> [pkg...]

pkg supports either:
  - std package path, e.g. container/ring
  - test package path, e.g. ./test/std/container/ring

Environment:
  LLGO_TEST_TIMEOUT  llgo test timeout (default: 10m)
  DOCKER_ARCHES      docker arches (default: "amd64 arm64")
EOF
	exit 2
}

if [[ $# -lt 1 ]]; then
	usage
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

timeout="${LLGO_TEST_TIMEOUT:-10m}"
docker_arches="${DOCKER_ARCHES:-amd64 arm64}"

normalize_pkg() {
	local in="$1"
	local std_pkg test_pkg
	case "$in" in
	./test/std/*)
		test_pkg="$in"
		std_pkg="${in#./test/std/}"
		;;
	test/std/*)
		test_pkg="./$in"
		std_pkg="${in#test/std/}"
		;;
	*)
		std_pkg="${in#./}"
		test_pkg="./test/std/$std_pkg"
		;;
	esac
	printf '%s %s\n' "$std_pkg" "$test_pkg"
}

for input_pkg in "$@"; do
	read -r std_pkg test_pkg <<<"$(normalize_pkg "$input_pkg")"

	echo
	echo "==== package: $std_pkg ($test_pkg) ===="

	echo "==> host: go test $test_pkg"
	go test "$test_pkg"

	echo "==> host: ./dev/llgo.sh test -timeout=$timeout $test_pkg"
	./dev/llgo.sh test -timeout="$timeout" "$test_pkg"

	echo "==> host: go run ./cmd/chore/check_std_symbols -pkg $std_pkg"
	go run ./cmd/chore/check_std_symbols -pkg "$std_pkg"

	log_dir="$(mktemp -d)"
	pids=()
	for arch in $docker_arches; do
		log="$log_dir/${arch}.log"
		(
			set -euo pipefail
			echo "==> docker[$arch]: go test $test_pkg"
			./dev/docker.sh "$arch" bash -lc "go test $test_pkg"
			echo "==> docker[$arch]: ./dev/llgo.sh test -timeout=$timeout $test_pkg"
			./dev/docker.sh "$arch" bash -lc "./dev/llgo.sh test -timeout=$timeout $test_pkg"
		) >"$log" 2>&1 &
		pids+=("$arch:$!:$log")
	done

	docker_failed=0
	for item in "${pids[@]}"; do
		arch="${item%%:*}"
		rest="${item#*:}"
		pid="${rest%%:*}"
		log="${rest#*:}"
		if wait "$pid"; then
			echo "==> docker[$arch]: PASS"
			cat "$log"
		else
			docker_failed=1
			echo "==> docker[$arch]: FAIL"
			cat "$log"
		fi
	done
	rm -rf "$log_dir"
	if [[ "$docker_failed" -ne 0 ]]; then
		exit 1
	fi
done

echo
echo "All checks passed."
