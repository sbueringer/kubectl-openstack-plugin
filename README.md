
# Kubectl Openstack Plugin

[![Travis](https://img.shields.io/travis/sbueringer/kubectl-openstack-plugin.svg)](https://travis-ci.org/sbueringer/kubectl-openstack-plugin)[![Codecov](https://img.shields.io/codecov/c/github/sbueringer/kubectl-openstack-plugin.svg)](https://codecov.io/gh/sbueringer/kubectl-openstack-plugin)[![CodeFactor](https://www.codefactor.io/repository/github/sbueringer/kubectl-openstack-plugin/badge)](https://www.codefactor.io/repository/github/sbueringer/kubectl-openstack-plugin)[![GoReportCard](https://goreportcard.com/badge/github.com/sbueringer/kubectl-openstack-plugin?style=plastic)](https://goreportcard.com/report/github.com/sbueringer/kubectl-openstack-plugin)[![GitHub release](https://img.shields.io/github/release/sbueringer/kubectl-openstack-plugin.svg)](https://github.com/sbueringer/kubectl-openstack-plugin/releases)

based on https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/sample-cli-plugin

TODO:
* enable output via go template like json path (from both openstack & kube object)
* unit tests

Known Issues
* flags only work with kubectl 1.12 (no plugin.yaml anymore)
