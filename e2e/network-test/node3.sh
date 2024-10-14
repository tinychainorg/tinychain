#!/bin/bash
set -ex

./scripts/build.sh

BASEDIR=$(dirname "$0")
./build/tinychain-darwin-arm64 node -db $BASEDIR/data/node3.db -port 8082 -peers http://0.0.0.0:8080 -miner
