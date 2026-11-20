#!/usr/bin/env bash

# Shared helpers for selecting exact Go toolchains in local and CI scripts.
# This file is intended to be sourced.

llgo_supported_go_releases() {
	printf '%s\n' '1.20 1.21 1.22 1.23 1.24 1.25 1.26 1.27'
}

llgo_resolve_go_version() {
	local repo_root=$1
	local requested=$2
	case "${requested}" in
		1.20) printf '%s\n' 1.20.14 ;;
		1.21) printf '%s\n' 1.21.13 ;;
		1.22) printf '%s\n' 1.22.12 ;;
		1.23) printf '%s\n' 1.23.12 ;;
		1.24) printf '%s\n' 1.24.13 ;;
		1.25) printf '%s\n' 1.25.11 ;;
		1.26) printf '%s\n' 1.26.7 ;;
		1.27) tr -d '[:space:]' <"${repo_root}/.go-version" ;;
		*)
			if [[ "${requested}" =~ ^1\.2[0-7]\.[0-9]+$ ]]; then
				printf '%s\n' "${requested}"
			else
				return 2
			fi
			;;
	esac
}

llgo_go_root() {
	local version=$1
	local current_version
	local toolchain_root

	current_version="$(GOTOOLCHAIN=local go env GOVERSION 2>/dev/null || true)"
	if [[ "${current_version}" == "go${version}" ]]; then
		toolchain_root="$(GOTOOLCHAIN=local go env GOROOT)"
	else
		toolchain_root="$(GOTOOLCHAIN="go${version}" go env GOROOT)"
	fi
	if [[ "${OS:-}" == "Windows_NT" ]] && command -v cygpath >/dev/null 2>&1; then
		toolchain_root="$(cygpath -u "${toolchain_root}")"
	fi

	local go_binary
	go_binary="$(llgo_go_binary "${toolchain_root}")"
	if [[ ! -x "${go_binary}" ]]; then
		echo "missing go binary for go${version}: ${go_binary}" >&2
		return 1
	fi
	if [[ "$(GOTOOLCHAIN=local "${go_binary}" env GOVERSION)" != "go${version}" ]]; then
		echo "failed to select exact Go toolchain go${version}" >&2
		return 1
	fi
	printf '%s\n' "${toolchain_root}"
}

llgo_go_binary() {
	local toolchain_root=$1
	if [[ "${OS:-}" == "Windows_NT" ]]; then
		printf '%s/bin/go.exe\n' "${toolchain_root}"
	else
		printf '%s/bin/go\n' "${toolchain_root}"
	fi
}
