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


set +e

go_test_args=("-race")

# test v1
gotestsum --packages="./" \
    --junitfile sponge_log.xml \
    --format standard-verbose \
    -- "${go_test_args[@]}" 2>&1 | tee sponge_log.log
exit_code=$(($exit_code + $?))


# switch to v2 and test

cd v2
set -e
try3 go mod download
set +e

gotestsum --packages="./" \
    --junitfile sponge_log.xml \
    --format standard-verbose \
    -- "${go_test_args[@]}" 2>&1 | tee sponge_log.log
exit_code=$(($exit_code + $?))

# Send logs to Flaky Bot for continuous builds.
if [[ $KOKORO_BUILD_ARTIFACTS_SUBDIR = *"continuous"* ]]; then
  cd ..
  chmod +x $KOKORO_GFILE_DIR/linux_amd64/flakybot
  $KOKORO_GFILE_DIR/linux_amd64/flakybot
fi

exit $exit_code
