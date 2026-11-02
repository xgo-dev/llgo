#!/usr/bin/env bash

# The quoted commands and startup fixtures must expand in the child shell.
# shellcheck disable=SC2016

set -euo pipefail

repository="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temporary="$(mktemp -d "${TMPDIR:-/tmp}/llgo-shell-test.XXXXXXXX")"
temporary="$(cd "$temporary" && pwd)"
trap 'rm -rf "$temporary"' EXIT

fail() {
    printf 'installer shell test: %s\n' "$*" >&2
    exit 1
}

# Distro CI bounds each startup separately so a stuck shell still reports its
# captured output. Leave local and platform tests free of a coreutils dependency.
timeout_binary=""
if [[ -n "${LLGO_SHELL_TEST_TIMEOUT:-}" ]]; then
    timeout_binary="$(command -v timeout || command -v gtimeout)" || fail 'GNU timeout is required'
fi

# Use the real piped entrypoint and archive/link installation, without network
# downloads or a compiler build. Release CI separately executes real archives.
mkdir -p "$temporary/payload/bin" "$temporary/payload/runtime"
printf '#!/bin/sh\nprintf "llgo installer fixture\\n"\n' >"$temporary/payload/bin/llgo"
chmod +x "$temporary/payload/bin/llgo"
printf 'module github.com/xgo-dev/llgo/runtime\n' >"$temporary/payload/runtime/go.mod"
tar -czf "$temporary/llgo.tar.gz" -C "$temporary/payload" .

install_fixture() {
    "${clean_env[@]}" LLGO_VERSION=1.2.3 LLGO_INSTALL_DEPS=0 \
        LLGO_UPDATE_PATH="${update_path:-1}" LLGO_INSTALL_ROOT="$install_root" \
        LLGO_ARCHIVE_PATH="$temporary/llgo.tar.gz" \
        /bin/bash <"$repository/install.sh" >"$temporary/install.log"
}

check_new_shell() {
    local mode
    local assertion='
        test "$(command -v llgo)" = "$LLGO_TEST_BIN/llgo" || exit 1
        test "$(llgo version)" = "llgo installer fixture" || exit 1
        test "$(printf "%s\n" "$PATH" | tr : "\n" | grep -Fxc "$LLGO_TEST_BIN")" = 1
    '
    if [[ "$shell_name" == fish ]]; then
        assertion='
            test (command -s llgo) = "$LLGO_TEST_BIN/llgo"; or exit 1
            test (llgo version) = "llgo installer fixture"; or exit 1
            test (printf "%s\n" $PATH | grep -Fxc "$LLGO_TEST_BIN") = 1
        '
    fi
    for mode in -ic -lic; do
        # POSIX sh/dash read .profile only on login; unlike bash/zsh/fish there
        # is no standard per-user rc file for a fresh non-login shell.
        if [[ "$shell_name" == sh || "$shell_name" == dash ]] && [[ "$mode" == -ic ]]; then
            continue
        fi
        printf 'START %s %s (%s)\n' "$shell_name" "$mode" "$scenario"
        local startup=("${clean_env[@]}" LLGO_TEST_BIN="$install_root/bin"
            "$shell_binary" "$mode" "$assertion")
        if [[ -n "$timeout_binary" ]]; then
            startup=("$timeout_binary" --verbose -k 2 "$LLGO_SHELL_TEST_TIMEOUT" "${startup[@]}")
        fi
        if "${startup[@]}" </dev/null >"$temporary/shell.log" 2>&1; then
            :
        else
            local status=$?
            cat "$temporary/install.log" "$temporary/shell.log" >&2
            fail "$shell_name $mode ($scenario) did not find exactly one installed PATH entry (exit $status)"
        fi
    done
}

[[ "$#" -gt 0 ]] || set -- bash zsh fish sh
for shell_name in "$@"; do
    shell_binary="$(command -v "$shell_name")" || fail "$shell_name is required"
    scenarios=(default)
    case "$shell_name" in
        bash) scenarios+=(bash_login bash_profile source_bashrc) ;;
        zsh) scenarios+=(zdotdir legacy_zprofile user_zprofile) ;;
        fish) scenarios+=(xdg_config) ;;
    esac
    for scenario in "${scenarios[@]}"; do
        printf 'START installer for %s (%s)\n' "$shell_name" "$scenario"
        test_home="$temporary/$shell_name-$scenario"
        mkdir -p "$test_home"
        # Never inherit the installer's PATH: a fresh terminal must find llgo
        # solely by loading its own real startup files. No user files are read.
        clean_env=(env -i HOME="$test_home" SHELL="$shell_binary"
            PATH=/usr/bin:/bin:/usr/sbin:/sbin TERM=dumb)
        case "$scenario" in
            bash_login)
                printf '# Existing user login configuration\n' >"$test_home/.bash_login"
                ;;
            bash_profile)
                printf '# Existing user login configuration\n' >"$test_home/.bash_profile"
                ;;
            source_bashrc)
                printf '. "$HOME/.bashrc"\n' >"$test_home/.bash_profile"
                ;;
            zdotdir) clean_env+=(ZDOTDIR="$test_home/zsh config") ;;
            legacy_zprofile)
                printf 'export PATH=/old/llgo/bin:$PATH # LLGo installer\n' >"$test_home/.zprofile"
                ;;
            user_zprofile)
                printf '# User login configuration\n' >"$test_home/.zprofile"
                ;;
            xdg_config) clean_env+=(XDG_CONFIG_HOME="$test_home/xdg config") ;;
        esac
        # A lower-priority .profile must not be hidden by a newly created
        # .bash_profile, or modified when a higher-priority file already exists.
        printf 'export LLGO_TEST_PROFILE=preserved\n' >"$test_home/.profile"
        install_root="$test_home/LLGo 'quote' \$dollar [glob] \\path"
        install_fixture
        install_fixture
        check_new_shell
        install_root="$test_home/other root"
        install_fixture
        check_new_shell
        if [[ "$shell_name" == bash ]]; then
            if [[ "$scenario" == default ]]; then
                [[ ! -e "$test_home/.bash_profile" && ! -e "$test_home/.bash_login" ]] ||
                    fail 'installer hid an existing .profile'
                grep -Fqx 'export LLGO_TEST_PROFILE=preserved' "$test_home/.profile" ||
                    fail 'installer removed user configuration'
            else
                [[ "$(<"$test_home/.profile")" == 'export LLGO_TEST_PROFILE=preserved' ]] ||
                    fail 'installer changed an inactive login profile'
            fi
        fi
        if [[ "$scenario" == legacy_zprofile ]]; then
            ! grep -Fq '/old/llgo/bin' "$test_home/.zprofile" || fail 'stale zprofile entry survived'
        elif [[ "$scenario" == user_zprofile ]]; then
            [[ "$(<"$test_home/.zprofile")" == '# User login configuration' ]] ||
                fail 'installer modified an unrelated zprofile'
        fi
        printf 'PASS %s (%s): fresh shells, idempotence, and changed installation root\n' "$shell_name" "$scenario"
    done
done

# Disabling profile updates must not create any startup files, even though the
# current installer process and GITHUB_PATH still receive the executable path.
test_home="$temporary/disabled"
mkdir -p "$test_home"
clean_env=(env -i HOME="$test_home" SHELL=/bin/zsh PATH=/usr/bin:/bin:/usr/sbin:/sbin)
install_root="$temporary/disabled-root"
update_path=0
install_fixture
[[ -z "$(ls -A "$test_home")" ]] || fail 'LLGO_UPDATE_PATH=0 modified shell configuration'
printf 'PASS LLGO_UPDATE_PATH=0\n'

update_path=1
clean_env=(env -i HOME="$test_home" SHELL=/bin/unknown-shell PATH=/usr/bin:/bin:/usr/sbin:/sbin)
install_fixture
[[ -z "$(ls -A "$test_home")" ]] || fail 'unknown shell received incompatible configuration'
grep -Fq 'not configured automatically' "$temporary/install.log" || fail 'unknown shell was not reported'
printf 'PASS unsupported shell warning\n'

clean_env=(env -i HOME="$test_home" SHELL= PATH=/usr/bin:/bin:/usr/sbin:/sbin)
install_fixture
[[ -f "$test_home/.profile" ]] || fail 'empty SHELL did not fall back to .profile'
printf 'PASS empty SHELL fallback\n'
