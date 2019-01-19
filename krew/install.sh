#!/usr/bin/env bash


kubectl krew install --manifest=./krew/os.yaml --archive=bazel-bin/cmd/kubectl-os/kubectl_os_tar.tar.gz
#kubectl krew install --manifest=foo.yaml --archive=foo.tar.gz