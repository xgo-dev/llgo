#!/usr/bin/env bash
set -euo pipefail
sudo tee /etc/yum.repos.d/goplus.repo <<'EOF'
[goplus]
name=XGo
baseurl=https://pkgs.xgo.dev/yum/
enabled=1
gpgcheck=0
EOF
sudo yum install -y llgo
