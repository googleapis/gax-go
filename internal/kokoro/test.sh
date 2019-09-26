#!/bin/bash

# Fail on any error
set -eo

# Display commands being run
set -x

# cd to project dir on Kokoro instance
cd github/gax-go

go version

# Set $GOPATH
export GOPATH="$HOME/go"
export GAX_HOME=$GOPATH/src/github.com/googleapis/gax-go
export PATH="$GOPATH/bin:$PATH"
export GO111MODULE=on
mkdir -p $GAX_HOME

# Move code into $GOPATH and get dependencies
git clone . $GAX_HOME
cd $GAX_HOME

try3() { eval "$*" || eval "$*" || eval "$*"; }

# All packages, including +build tools, are fetched.
try3 go mod download
./internal/kokoro/vet.sh
go test -race -v . 2>&1 | tee $KOKORO_ARTIFACTS_DIR/$KOKORO_GERRIT_CHANGE_NUMBER.txt

cd v2
try3 go mod download
go test -race -v . 2>&1 | tee $KOKORO_ARTIFACTS_DIR/$KOKORO_GERRIT_CHANGE_NUMBER.txt
