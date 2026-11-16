#!/usr/bin/env bash
set -euo pipefail
sudo pacman -Syu --needed --noconfirm base-devel ca-certificates curl pkgconf gc openssl zlib \
  libffi cjson sqlite libuv
