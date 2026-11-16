#!/usr/bin/env bash
set -euo pipefail
sudo dnf install -y gcc gcc-c++ make ca-certificates curl pkgconf-pkg-config \
  gc-devel openssl-devel zlib-devel libffi-devel cjson-devel sqlite-devel libuv-devel
