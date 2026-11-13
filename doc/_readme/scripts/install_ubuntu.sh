# shellcheck disable=all
set -eu
llvm_key="$(mktemp)"
trap 'rm -f "$llvm_key"' EXIT
wget -4 --retry-on-host-error --tries=3 --timeout=15 --waitretry=2 -O "$llvm_key" https://apt.llvm.org/llvm-snapshot.gpg.key
sudo install -m 0644 "$llvm_key" /usr/share/keyrings/llvm-snapshot.asc
echo "deb [signed-by=/usr/share/keyrings/llvm-snapshot.asc] https://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-22 main" | sudo tee /etc/apt/sources.list.d/llvm.list
sudo apt-get update
sudo apt-get install -y llvm-22-dev clang-22 libclang-22-dev lld-22 lldb-22 libunwind-22-dev libc++-22-dev pkg-config libgc-dev libssl-dev zlib1g-dev libffi-dev libcjson-dev libsqlite3-dev libuv1-dev
sudo apt-get install -y python3-dev # optional
./install.sh
