#!/usr/bin/env bash

docker run -it -w /github/workspace -v $(pwd):/github/workspace -v /tmp:/tmp --entrypoint bazel sbueringer/bazel build //cmd/kubectl-openstack:kubectl-openstack //cmd/kubectl-openstack:kubectl-openstack_tar
