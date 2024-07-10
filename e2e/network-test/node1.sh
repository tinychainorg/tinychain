#!/bin/bash
set -ex

./scripts/build.sh

BASEDIR=$(dirname "$0")
./build/tinychain-darwin-arm64 node -db $BASEDIR/data/node1.db -port 8080 -miner --miner-tag $1
#  -explorer
