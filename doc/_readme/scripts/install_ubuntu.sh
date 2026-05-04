# shellcheck disable=all
echo "deb http://apt.llvm.org/$(lsb_release -cs)/ llvm-toolchain-$(lsb_release -cs)-19 main" | sudo tee /etc/apt/sources.list.d/llvm.list
wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y llvm-19-dev clang-19 libclang-19-dev lld-19 libunwind-19-dev libc++-19-dev pkg-config libgc-dev libssl-dev zlib1g-dev libcjson-dev libsqlite3-dev libuv1-dev
sudo apt-get install -y python3.12-dev # optional
#curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
