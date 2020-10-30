#!/usr/bin/env sh
set -e

rm -rf build
mkdir build
go build -o build/maestro config.go context.go frontend.go healthcheck.go logging.go maestro.go process.go service.go
pushd frontend
yarn build
popd
cp -r frontend/dist build/frontend
