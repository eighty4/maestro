#!/bin/bash

rm -rf dist

go build -o dist/maestro \
  config.go \
  context.go \
  frontend.go \
  git.go \
  healthcheck.go \
  logging.go \
  maestro.go \
  process.go \
  service.go
