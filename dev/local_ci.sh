#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[ -f "$repo_root/go.mod" ] && [ -f "$repo_root/dev/docker-compose.yml" ] || {
	echo "error: dev/local_ci.sh must live inside LLGO_ROOT (missing go.mod or dev/docker-compose.yml)" >&2
	exit 2
}
tmp_root="$(mktemp -d)"
workdir="$repo_root"

cleanup() {
	rm -rf "$tmp_root"
}
trap cleanup EXIT

export LLGO_ROOT="$workdir"

log_section() {
	printf "\n==== %s ====\n" "$1"
}

setup_python3_embed_pc() {
	# Some packages expect `python3-embed.pc`; create a shim that
	# points at whichever python-X.Y-embed pkg-config file exists.
	if pkg-config --exists python3-embed; then
		return
	fi

	local candidates=()
	if command -v python3 >/dev/null 2>&1; then
		local detected
		detected="$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")' 2>/dev/null || true)"
		[ -n "$detected" ] && candidates+=("$detected")
	fi
	candidates+=("3.13" "3.12" "3.11" "3.10")

	local chosen=""
	for ver in "${candidates[@]}"; do
		local pkg="python-${ver}-embed"
		if pkg-config --exists "$pkg"; then
			chosen="$pkg"
			break
		fi
	done
	if [ -z "$chosen" ]; then
		return
	fi

	local pcdir="$tmp_root/pc"
	mkdir -p "$pcdir"
	local libdir
	libdir=$(pkg-config --variable=libdir "$chosen")
	ln -sf "$libdir/pkgconfig/${chosen}.pc" "$pcdir/python3-embed.pc"
	export PKG_CONFIG_PATH="$pcdir${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
}

setup_python3_embed_pc

ensure_llama2_model() {
	local dst="$workdir/_demo/c/llama2-c/stories15M.bin"
	if [ -f "$dst" ]; then
		return
	fi

	local assets_dir="${LLGO_ASSETS_DIR:-/opt/llgo-assets}"
	local src="$assets_dir/llama2-c/stories15M.bin"
	if [ -f "$src" ]; then
		mkdir -p "$(dirname "$dst")"
		cp -f "$src" "$dst"
		return
	fi

	mkdir -p "$(dirname "$dst")"
	local url="${LLGO_LLAMA2_URL:-https://huggingface.co/karpathy/tinyllamas/resolve/main/stories15M.bin}"
	local tmp="${dst}.tmp.$$"
	if command -v curl >/dev/null 2>&1; then
		if ! curl -fL "$url" -o "$tmp"; then
			rm -f "$tmp"
			exit 1
		fi
	elif command -v wget >/dev/null 2>&1; then
		if ! wget -O "$tmp" "$url"; then
			rm -f "$tmp"
			exit 1
		fi
	else
		echo "error: missing llama2 model and neither curl nor wget is available: $dst" >&2
		exit 1
	fi
	mv -f "$tmp" "$dst"
}

ensure_llama2_model

run_hello() {
	local mod_version="$1"
	local go_version=1.26
	if [[ "$mod_version" == 1.20 ]]; then
		go_version=1.20
	fi
	LLGO="$(command -v llgo)" LLGO_HELLO_EMBED=0 \
		"$repo_root/dev/with_go_version.sh" "$go_version" \
		"$repo_root/dev/test_helloworld.sh" "$mod_version"
}

log_section "Format"
for dir in . runtime; do
	pushd "$workdir/$dir" >/dev/null
	fmt_output="$(
		find . -name '*.go' -type f ! -name 'xgo_autogen.go' -print0 \
			| xargs -0 gofmt -l \
			| sed 's|^\\./||' \
			|| true
	)"
	if [ -n "$fmt_output" ]; then
		printf "Detected gofmt differences in %s:\n%s\n" "$dir" "$fmt_output"
		exit 1
	fi
	popd >/dev/null
done

log_section "Go Build"
(cd "$workdir" && go build ./...)

log_section "Go Test"
(cd "$workdir" && go test -timeout 30m ./...)

log_section "Runtime module with Go 1.20 and Go 1.26"
(cd "$workdir" && dev/test_runtime_go_versions.sh)

log_section "Install llgo"
(cd "$workdir" && go install -tags=dev ./cmd/llgo)
gobin="$(cd "$workdir" && go env GOBIN)"
if [ -z "$gobin" ]; then
	gopath_raw="$(cd "$workdir" && go env GOPATH)"
	gobin="${gopath_raw%%:*}/bin"
fi
export PATH="$gobin:$PATH"

log_section "llgo test with Go 1.20 through Go 1.26"
(cd "$workdir" && LLGO="$(command -v llgo)" dev/test_go_versions.sh)

log_section "Demo Tests"
demo_jobs="${LLGO_DEMO_JOBS:-}"
if [ -z "$demo_jobs" ]; then
	if command -v nproc >/dev/null 2>&1; then
		demo_jobs="$(nproc)"
	elif command -v getconf >/dev/null 2>&1; then
		demo_jobs="$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1)"
	else
		demo_jobs=1
	fi
	if [ "$demo_jobs" -gt 4 ]; then
		demo_jobs=4
	fi
fi
(cd "$workdir" && LLGO_DEMO_JOBS="$demo_jobs" bash .github/workflows/test_demo.sh)

log_section "Build targets"
(cd "$workdir/_demo/embed/targetsbuild" && bash build.sh)

log_section "Hello World"
hello_logs=()
for mod in 1.20 1.21 1.22 1.23 1.24 1.26; do
	log="$tmp_root/hello-${mod}.log"
	(run_hello "$mod") >"$log" 2>&1 &
	hello_logs+=("$log:$!")
done
for item in "${hello_logs[@]}"; do
	log="${item%%:*}"
	pid="${item##*:}"
	if ! wait "$pid"; then
		cat "$log"
		exit 1
	fi
done

log_section "Build Cache"
(cd "$workdir/test/buildcache" && bash test.sh)

log_section "Done"
