#!/usr/bin/env bash

cd ${GITHUB_HOME}/kubectl-openstack-plugin

if [ "${KUBECTL_PLUGINS_GLOBAL_FLAG_V}" == "8" ]
then
    env | grep KUBE
fi

bazel run //:gazelle
bazel run //cmd/kubectl-os:kubectl-os -- "$@"
