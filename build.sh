#!/usr/bin/env bash

docker run -it -w /github/workspace -v $(pwd):/github/workspace --entrypoint bazel sbueringer/bazel build //cmd/kubectl-os:kubectl-os