#!/usr/bin/env bash
set -euo pipefail
echo 'deb [trusted=yes] https://pkgs.xgo.dev/apt/ /' | sudo tee /etc/apt/sources.list.d/goplus.list
sudo apt-get update
sudo apt-get install -y llgo
