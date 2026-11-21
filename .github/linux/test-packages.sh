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
        # Enable the LLVM 23 repository inside the clean Ubuntu image.
        image=ubuntu:26.04
    else
        image=fedora:45
    fi
    docker run --rm --platform "linux/$arch" \
        -e DEBIAN_FRONTEND=noninteractive \
        -v "${package[0]}:/packages/llgo.$format:ro" \
        -v "$tests:/tests:ro" \
        "$image" bash -c '
            set -euo pipefail
            if [[ "$1" == deb ]]; then
                apt-get update
                apt-get install -y --no-install-recommends ca-certificates curl
                curl -fsSL --retry 5 https://apt.llvm.org/llvm-snapshot.gpg.key -o /usr/share/keyrings/llvm-snapshot.asc
                . /etc/os-release
                echo "deb [signed-by=/usr/share/keyrings/llvm-snapshot.asc] https://apt.llvm.org/$VERSION_CODENAME/ llvm-toolchain-$VERSION_CODENAME-23 main" > /etc/apt/sources.list.d/llvm.list
                apt-get update
                apt-get install -y --no-install-recommends /packages/llgo.deb
            else
                dnf install -y /packages/llgo.rpm
            fi
            bash /tests/test-installed.sh
        ' bash "$format"
done
