#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '[goplus]' 'name=XGo' 'baseurl=https://pkgs.xgo.dev/yum/' 'enabled=1' 'gpgcheck=0' | \
  sudo tee /etc/yum.repos.d/goplus.repo
sudo yum install -y llgo
