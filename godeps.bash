#!/bin/bash

cd $(bazel info workspace)

#echo "running dep ensure.."
#dep ensure
echo "updating go modules and vendor directory.."
# Enable Go modules.
export GO111MODULE=on
go mod tidy
go mod vendor

echo "recreating BUILD files and running gazelle.."
find vendor -name 'BUILD*' -print0 | xargs -0 rm
bazel run //:gazelle
