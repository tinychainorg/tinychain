#!/bin/bash
set -ex

./scripts/build.sh

BASEDIR=$(dirname "$0")
./build/tinychain-darwin-arm64 node -db $BASEDIR/data/node2.db -port 8081 -peers http://0.0.0.0:8080