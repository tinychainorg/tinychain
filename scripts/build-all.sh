set -ex

# brew install filosottile/musl-cross/musl-cross
# brew install llvm
# brew tap messense/macos-cross-toolchains
# brew install aarch64-linux-gnu
# brew install aarch64-unknown-linux-gnu

export CGO_ENABLED=1
export CGO_CFLAGS="-D_LARGEFILE64_SOURCE"

cd cli/

# target: mac arm64 (apple silicon)
export GOOS=darwin
export GOARCH=arm64
unset CC
unset CXX
go build -o tinychain-$GOOS-$GOARCH
mv tinychain-$GOOS-$GOARCH ../build/

# target: mac amd64 (intel)
export GOOS=darwin
export GOARCH=amd64
unset CC
unset CXX
go build -o tinychain-$GOOS-$GOARCH
mv tinychain-$GOOS-$GOARCH ../build/

# target: linux arm64
export GOOS=linux 
export GOARCH=arm64
export CC=aarch64-linux-gnu-gcc
export CXX=aarch64-linux-musl-g++
go build -o tinychain-$GOOS-$GOARCH -ldflags "-linkmode external -extldflags -static"
mv tinychain-$GOOS-$GOARCH ../build/

# target: linux amd64
export GOOS=linux 
export GOARCH=amd64
export CC=x86_64-linux-musl-gcc
export CXX=x86_64-linux-musl-g++
go build -o tinychain-$GOOS-$GOARCH -ldflags "-linkmode external -extldflags -static"
mv tinychain-$GOOS-$GOARCH ../build/
