#!/bin/bash

# Fail on any error
set -eo

# Display commands being run
set -x

# Only run the linter on go1.13, since it needs type aliases (and we only care about its output once).
# TODO(deklerk) We should pass an environment variable from kokoro to decide this logic instead.
if [[ `go version` != *"go1.13"* ]]; then
    exit 0
fi

go install \
  github.com/golang/protobuf/proto \
  github.com/golang/protobuf/protoc-gen-go \
  golang.org/x/lint/golint \
  golang.org/x/tools/cmd/goimports \
  honnef.co/go/tools/cmd/staticcheck

# Look at all .go files (ignoring .pb.go files) and make sure they have a Copyright. Fail if any don't.
find . -type f -name "*.go" ! -name "*.pb.go" -exec grep -L "\(Copyright [0-9]\{4,\}\)" {} \; 2>&1 | tee /dev/stderr | (! read)

# Fail if a dependency was added without the necessary go.mod/go.sum change
# being part of the commit.
go mod tidy
pushd v2
  go mod tidy
popd
git diff go.mod | tee /dev/stderr | (! read)
git diff go.sum | tee /dev/stderr | (! read)

# Easier to debug CI.
pwd

gofmt -s -d -l . 2>&1 | tee /dev/stderr | (! read)
goimports -l . 2>&1 | tee /dev/stderr | (! read)

golint ./... 2>&1 | tee /dev/stderr | (! read)
staticcheck ./...
