# shellcheck disable=all
brew update
brew install llvm@21 lld@21 bdw-gc openssl cjson libffi libuv pkg-config
brew install python@3.12 # optional
brew link --overwrite llvm@21 lld@21 libffi
# curl https://raw.githubusercontent.com/xgo-dev/llgo/refs/heads/main/install.sh | bash
./install.sh
