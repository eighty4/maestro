#!/bin/sh
set -e

# run through all the checks done for ci

# only exit error when fmt or tidy changes source if working directory pristine
_git_status_output=$(git status --porcelain)

(cd www && pnpm build)

if ! command -v "golangci-lint" >/dev/null 2>&1; then
  echo '*** install golangci-lint'
  brew tap golangci/tap
  brew install golangci/tap/golangci-lint
else
  echo '*** check for golangci-lint upgrade'
  brew upgrade golangci/tap/golangci-lint
fi

echo '\n*** build ***'
go build

echo '\n*** fmt/tidy composable ***'
(cd composable && go mod tidy)
(cd composable && go fmt .)
echo '\n*** fmt/tidy git ***'
(cd git && go mod tidy)
(cd git && go fmt .)
echo '\n*** fmt/tidy testutil ***'
(cd testutil && go mod tidy)
(cd testutil && go fmt .)
echo '\n*** fmt/tidy util ***'
(cd util && go mod tidy)
(cd util && go fmt .)
echo '\n*** fmt/tidy . ***'
go mod tidy
go fmt .

if [ -z "$_git_status_output" ]; then
  echo "check go mod tidy and fmt changes were made"
  git diff --exit-code
fi

echo '\n*** lint composable ***'
(cd composable && golangci-lint run)
echo '\n*** lint git ***'
(cd git && golangci-lint run)
echo '\n*** lint testutil ***'
(cd testutil && golangci-lint run)
echo '\n*** lint util ***'
(cd util && golangci-lint run)
echo '\n*** lint . ***'
golangci-lint run

echo '\n*** test ***'
go list -f '{{.Dir}}' -m | xargs go test

echo '\n*** integration test ***'
PATH="$PATH:$(pwd)"
(cd integration/rust && cargo build)
(cd integration && go run integration.go)

if [ -n "$_git_status_output" ]; then
  echo
  echo "all ci verifications passed"
  echo "however, working directory had uncommited changes before running go fmt and go mod tidy"
  exit 1
fi
