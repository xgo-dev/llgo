#!/usr/bin/env bash

# Shared helpers for selecting exact Go toolchains in local and CI scripts.
# This file is intended to be sourced.

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
		1.26) tr -d '[:space:]' <"${repo_root}/.go-version" ;;
		*)
			if [[ "${requested}" =~ ^1\.2[0-6]\.[0-9]+$ ]]; then
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

	if [[ ! -x "${toolchain_root}/bin/go" ]]; then
		echo "missing go binary for go${version}: ${toolchain_root}/bin/go" >&2
		return 1
	fi
	if [[ "$(GOTOOLCHAIN=local "${toolchain_root}/bin/go" env GOVERSION)" != "go${version}" ]]; then
		echo "failed to select exact Go toolchain go${version}" >&2
		return 1
	fi
	printf '%s\n' "${toolchain_root}"
}
