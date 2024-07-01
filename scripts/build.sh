set -ex

# brew install filosottile/musl-cross/musl-cross
# brew install llvm
# brew tap messense/macos-cross-toolchains
# brew install aarch64-linux-gnu
# brew install aarch64-unknown-linux-gnu

export CGO_ENABLED=1
export CGO_CFLAGS="-D_LARGEFILE64_SOURCE"

mkdir -p build/

cd cli/

# target: mac arm64 (apple silicon)
export GOOS=darwin
export GOARCH=arm64
unset CC
unset CXX
go build -o ../build/tinychain-$GOOS-$GOARCH