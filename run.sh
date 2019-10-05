#!/usr/bin/env bash

cd ${GITHUB_HOME}/sbueringer/kubectl-openstack-plugin

if [ "${KUBECTL_PLUGINS_GLOBAL_FLAG_V}" == "8" ]
then
    env | grep KUBE
fi

go run ./cmd/kubectl-openstack  "$@"
