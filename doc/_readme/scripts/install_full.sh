# shellcheck disable=all
git clone https://github.com/xgo-dev/llgo.git
cd llgo
go install -v ./cmd/...
go install -v ./chore/...  # compile all tools except pydump
export LLGO_ROOT=$PWD
cd _xtool
llgo install ./...   # compile pydump
go install github.com/goplus/hdq/chore/pysigfetch@v0.8.1  # compile pysigfetch
