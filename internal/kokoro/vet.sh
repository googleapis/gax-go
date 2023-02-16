#!/bin/bash

# Fail on any error
set -eo

# Display commands being run
set -x

if [[ $(go version) != *"go1.20"* ]]; then
  exit 0
fi

# Look at all .go files (ignoring .pb.go files) and make sure they have a Copyright. Fail if any don't.
find . -type f -name "*.go" ! -name "*.pb.go" -exec grep -L "\(Copyright [0-9]\{4,\}\)" {} \; 2>&1 | tee /dev/stderr | (! read)

# Fail if a dependency was added without the necessary go.mod/go.sum change
# being part of the commit.
go mod tidy
git diff go.mod | tee /dev/stderr | (! read)
git diff go.sum | tee /dev/stderr | (! read)

pushd v2
  go mod tidy
  git diff go.mod | tee /dev/stderr | (! read)
  git diff go.sum | tee /dev/stderr | (! read)
popd

# Easier to debug CI.
pwd

gofmt -s -d -l . 2>&1 | tee /dev/stderr | (! read)
goimports -l . 2>&1 | tee /dev/stderr | (! read)

golint ./... 2>&1 | tee /dev/stderr | (! read)
staticcheck ./...
