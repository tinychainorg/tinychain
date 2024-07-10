set -ex
./scripts/build.sh
./build/tinychain-darwin-arm64 node -db build/tinychain.db
