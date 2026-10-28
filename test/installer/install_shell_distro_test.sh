#!/usr/bin/env bash

set -euo pipefail

repository="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ "$#" == 0 ]]; then
    set -- ubuntu:24.04 debian:12 fedora:latest archlinux:latest alpine:latest opensuse/tumbleweed:latest
fi

# Only shell startup is tested here. Keep compiler builds, native dependency
# installation, and real release execution in their existing platform jobs.
status=0
for image in "$@"; do
    case "$image" in
        ubuntu:* | debian:*)
            setup='apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends bash zsh fish tar gzip'
            ;;
        fedora:*)
            setup='dnf install -y --setopt=install_weak_deps=False bash zsh fish tar gzip'
            ;;
        archlinux:*)
            setup='pacman -Syu --needed --noconfirm bash zsh fish tar gzip'
            ;;
        alpine:*)
            # BusyBox timeout only signals its direct child. Use the same
            # process-group watchdog as the other distributions.
            setup='apk add --no-cache bash zsh fish tar gzip coreutils'
            ;;
        opensuse/*:*)
            # These packages and their dependencies are in OSS. Unrelated
            # non-OSS and codec repositories must not gate shell startup tests.
            setup='sed -i s,http://,https://,g /etc/zypp/repos.d/*.repo && zypper --non-interactive refresh --repo repo-oss && zypper --non-interactive install --no-recommends --repo repo-oss bash zsh fish tar gzip'
            ;;
        *) printf 'Unsupported shell-test image: %s\n' "$image" >&2; exit 1 ;;
    esac
    printf '::group::Installer shell startup: %s\n' "$image"
    # The host deadline also covers Docker create/attach. Container-internal
    # timeouts cannot bound a stuck client or its teardown on their own.
    if python3 "$repository/test/installer/run_shell_container.py" "$image" "$setup"; then
        printf 'PASS %s\n' "$image"
    else
        exit_status=$?
        printf 'FAIL %s (exit %s)\n' "$image" "$exit_status" >&2
        if [[ "$exit_status" == 130 || "$exit_status" == 143 ]]; then
            printf '::endgroup::\n'
            exit "$exit_status"
        fi
        status=1
    fi
    printf '::endgroup::\n'
done
exit "$status"
