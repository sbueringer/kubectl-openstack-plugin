#!/usr/bin/env bash


kubectl krew install --manifest=./krew/openstack.yaml --archive=bazel-bin/cmd/kubectl-openstack/kubectl-openstack_tar.tar.gz
#kubectl krew install --manifest=./krew/openstack.yaml