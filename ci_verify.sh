#!/bin/sh
set -e

# run through all the checks done for ci

_git_status_output=$(git status --porcelain)

echo '\n*** cargo build ***'
cargo build --workspace

echo '\n*** cargo fmt -v ***'
cargo fmt --all -v
if [ -z "$_git_status_output" ]; then
  git diff --exit-code
fi

echo '\n*** cargo test ***'
cargo test --workspace

echo '\n*** cargo clippy -- -D warnings ***'
cargo clippy --all -- -D warnings

# echo '\n*** cargo run --example(s) ***'
# cargo run -p maestro_sync --example some_example

if [ -n "$_git_status_output" ]; then
  echo
  echo "all ci verifications passed"
  echo "however, working directory had uncommited changes before running cargo fmt"
  exit 1
fi
