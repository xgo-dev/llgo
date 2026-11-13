# syntax=docker/dockerfile:1

ARG BASE_IMAGE=golang:1.27.0-bookworm
FROM ${BASE_IMAGE} AS unpack

ARG TARGETPLATFORM

# GoReleaser supplies exactly one complete DEB per target platform.
COPY ${TARGETPLATFORM}/*.deb /tmp/llgo.deb
# Extract the payload so Go comes from the pinned base image, not Debian.
RUN dpkg-deb --extract /tmp/llgo.deb /install

FROM ${BASE_IMAGE}
# Keep native dependencies aligned with the DEB override in .goreleaser.yaml.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl gnupg \
    && curl -fsSL https://apt.llvm.org/llvm-snapshot.gpg.key \
        | gpg --dearmor -o /usr/share/keyrings/llvm.gpg \
    && . /etc/os-release \
    && echo "deb [signed-by=/usr/share/keyrings/llvm.gpg] https://apt.llvm.org/${VERSION_CODENAME}/ llvm-toolchain-${VERSION_CODENAME}-22 main" \
        > /etc/apt/sources.list.d/llvm-22.list \
    && apt-get update && apt-get install -y --no-install-recommends \
    libllvm22 llvm-22 clang-22 lld-22 libc++-22-dev tar xz-utils \
    build-essential pkg-config libgc-dev libssl-dev zlib1g-dev libffi-dev \
    libcjson-dev libsqlite3-dev libuv1-dev \
    && rm -rf /var/lib/apt/lists/*

COPY --from=unpack /install/ /
ENV LLVM_CONFIG=/usr/bin/llvm-config-22
WORKDIR /workspace
CMD ["llgo", "version"]
