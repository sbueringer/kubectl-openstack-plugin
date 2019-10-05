#!/usr/bin/env bash

WORKDIR=`echo $0 | sed -e s/build.sh//`
cd ${WORKDIR}

docker run --rm -v "$PWD":/usr/src/github.com/sbueringer/kubectl-openstack-plugin -w /usr/src/github.com/sbueringer/kubectl-openstack-plugin golang:1.13.1 go build -v -o ./kubectl-openstack ./cmd/kubectl-openstack
