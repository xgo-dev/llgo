#!/usr/bin/env bash
set -euo pipefail
sudo zypper --non-interactive install gcc gcc-c++ make ca-certificates curl pkg-config \
  gc-devel libopenssl-devel zlib-devel libffi-devel cJSON-devel sqlite3-devel libuv-devel
