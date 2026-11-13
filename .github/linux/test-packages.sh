#!/usr/bin/env bash
set -euo pipefail

packages=$(cd "${1:?package directory required}" && pwd)
arch=${2:?architecture required}
tests=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Fresh distributions keep the runner's installed LLVM and libraries from
# masking missing package contents or dependency declarations.
for format in deb rpm; do
    package=("$packages"/*.linux-"$arch"."$format")
    test "${#package[@]}" = 1
    test -s "${package[0]}"
    if [[ "$format" == deb ]]; then
        # LLVM 22 is in the standard Ubuntu 26.04 repository. Older releases
        # need apt.llvm.org enabled before installing the DEB.
        image=ubuntu:26.04
    else
        image=fedora:44
    fi
    docker run --rm --platform "linux/$arch" \
        -e DEBIAN_FRONTEND=noninteractive \
        -v "${package[0]}:/packages/llgo.$format:ro" \
        -v "$tests:/tests:ro" \
        "$image" bash -c '
            set -euo pipefail
            if [[ "$1" == deb ]]; then
                apt-get update
                apt-get install -y --no-install-recommends /packages/llgo.deb
            else
                dnf install -y /packages/llgo.rpm
            fi
            bash /tests/test-installed.sh
        ' bash "$format"
done
