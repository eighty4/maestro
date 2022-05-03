#!/bin/bash
set -e

./build.sh
pushd frontend
yarn
yarn build
popd
cp -r frontend/dist dist/frontend
