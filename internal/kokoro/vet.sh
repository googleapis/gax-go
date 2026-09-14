#!/bin/bash

# Fail on any error
set -eo

# Display commands being run
set -x

# In our latest version, do basic checks and module staleness checks.
if [[ $KOKORO_JOB_NAME == *"latest-version"* ]]; then

  # Look at all .go files (ignoring .pb.go files) and make sure they have a Copyright. Fail if any don't.
  find . -type f -name "*.go" ! -name "*.pb.go" -exec grep -L "\(Copyright [0-9]\{4,\}\)" {} \; 2>&1 | tee /dev/stderr | (! read)

  # Easier to debug CI.
  pwd

  # Execute tidy and linters inside every submodule (e.g. v2/)
  # This ensures that we don't just lint the root module.
  # It also fails if a dependency was added without the necessary go.mod/go.sum change.
  for mod in $(find . -name go.mod); do
    pushd $(dirname $mod)
    go mod tidy -diff
    gofmt -s -d -l . 2>&1 | tee /dev/stderr | (! read)
    goimports -l . 2>&1 | tee /dev/stderr | (! read)
    popd
  done
fi


# In our earliest version, do static checking
if [[ $KOKORO_JOB_NAME == *"earliest-version"* ]]; then
  staticcheck ./... 
fi
