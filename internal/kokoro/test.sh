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
export PATH="$GOPATH/bin:$PATH"
export GO111MODULE=on

try3() { eval "$*" || eval "$*" || eval "$*"; }

# All packages, including +build tools, are fetched.
try3 go mod download
./internal/kokoro/vet.sh

go get github.com/jstemmer/go-junit-report

set +e

go test -race -v . 2>&1 | tee sponge_log.log
cat sponge_log.log | go-junit-report -set-exit-code > sponge_log.xml
exit_code=$?

cd v2
set -e
try3 go mod download
set +e

go test -race -v . 2>&1 | tee sponge_log.log
cat sponge_log.log | go-junit-report -set-exit-code > sponge_log.xml
exit_code=$(($exit_code+$?))

# Send logs to the Build Cop Bot for continuous builds.
if [[ $KOKORO_BUILD_ARTIFACTS_SUBDIR = *"continuous"* ]]; then
  cd ..
  chmod +x $KOKORO_GFILE_DIR/linux_amd64/buildcop
  $KOKORO_GFILE_DIR/linux_amd64/buildcop
fi

exit $exit_code
