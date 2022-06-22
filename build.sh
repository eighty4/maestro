#!/bin/bash

rm -rf dist

go build -o dist/maestro \
  config.go \
  context.go \
  exec.go \
  frontend.go \
  git.go \
  gradle.go \
  healthcheck.go \
  logging.go \
  maestro.go \
  npm.go \
  process.go \
  service.go
