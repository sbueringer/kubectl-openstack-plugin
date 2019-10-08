
# Kubectl Openstack Plugin

[![Travis](https://img.shields.io/travis/sbueringer/kubectl-openstack-plugin.svg)](https://travis-ci.org/sbueringer/kubectl-openstack-plugin)[![Codecov](https://img.shields.io/codecov/c/github/sbueringer/kubectl-openstack-plugin.svg)](https://codecov.io/gh/sbueringer/kubectl-openstack-plugin)[![CodeFactor](https://www.codefactor.io/repository/github/sbueringer/kubectl-openstack-plugin/badge)](https://www.codefactor.io/repository/github/sbueringer/kubectl-openstack-plugin)[![GoReportCard](https://goreportcard.com/badge/github.com/sbueringer/kubectl-openstack-plugin?style=plastic)](https://goreportcard.com/report/github.com/sbueringer/kubectl-openstack-plugin)[![GitHub release](https://img.shields.io/github/release/sbueringer/kubectl-openstack-plugin.svg)](https://github.com/sbueringer/kubectl-openstack-plugin/releases)

based on [k8s.io/sample-cli-plugin](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/sample-cli-plugin)

# Installation 

## Prerequisites

* Afaik kubectl plugins without plugin.yaml only work with kubectl >=1.12

## Installation via go get

Just execute the following and make sure `$GOPATH/bin` is in your `$PATH`:
````
GO111MODULE=on go get github.com/sbueringer/kubectl-openstack-plugin/cmd/kubectl-openstack
````

Note: this currently only works without GO111MODULE=on because a replace directive for gophercloud is used. (see also: https://github.com/golang/go/issues/30354)

## Installation via download

Download the binary from [Releases](https://github.com/sbueringer/kubectl-openstack-plugin/releases) and place it in a directory in your `PATH`.


# Configuration

To access OpenStack via this plugin the OpenStack credentials must be configured either via env variables or via [clouds.yaml](https://docs.openstack.org/python-openstackclient/pike/configuration/index.html) config file.

## Configuration via Environment Variables

The plugin can be configured by setting the following env variables:
* `OS_USERNAME`
* `OS_PASSWORD`
* `OS_PROJECT_NAME` or `OS_TENANT_NAME`
* `OS_AUTH_URL`

## Configuration via config file

The location of the config file must be configured via `OPENSTACK_CONFIG_FILE` env var. An example `clouds.yaml`:
````
clouds:
  i01p015:
    auth:
      auth_url: http://192.168.122.10:35357/
      project_name: i01p015
      username: demo
      password: password
````

*Note*: The cloud/project_name is automatically discovered from the current kube context. E.g. a kube context named `i01p015-cluster-admin` leads to a cloud/project_name of `i01p015`. 
*Note*: The clouds.yaml file can be created from `.rc` files via the `import-config` sub command.

# Usage

The kubectl OpenStack plugin currently has three commands, which are shown here.

## kubectl openstack server

The `server` command combines information about Kubernetes Nodes with OpenStack Server.

````
$ kubectl openstack server
NODE_NAME               STATUS  KUBELET_VERSION  KUBEPROXY_VERSION  RUNTIME_VERSION  SERVER_ID                             STATE   CPU  RAM  IP
i01p015-kube-master01  Ready   v1.11.0           v1.11.0            docker://18.3.1  c11231ab-4315-4a77-b5fc-22f2a668d414  ACTIVE  2    15G  10.12.4.12
i01p015-kube-node01    Ready   v1.11.0           v1.11.0            docker://18.3.1  04acf401-dcf4-4e7c-8796-69662768067a  ACTIVE  2     8G  10.12.4.17
i01p015-kube-node02    Ready   v1.11.0           v1.11.0            docker://18.3.1  cf03414f-f692-4766-a797-16f01b154d6e  ACTIVE  2     8G  10.12.4.7
i01p015-kube-node03    Ready   v1.11.0           v1.11.0            docker://18.3.1  fca70123-2db0-430a-a84e-5010cc1f0f71  ACTIVE  2     8G  10.12.4.15
````

## kubectl openstack volumes

The `volumes` command combines information about Kubernetes Persistent Volumes & Nodes with OpenStack Volumes.

````
$ kubectl openstack volumes
CLAIM                                 PV_NAME                                   CINDER_ID                             SERVERS                 STATUS
default/cache                         pvc-15eb6f71-943a-11e8-9844-fa163e81bcc3  3c1e3f40-09ad-4a2a-b77e-8abc53f9d8d7  i01p015-kube-node02     in-use
monitoring/data-prometheus-0          pvc-02432937-93ed-11e8-9844-fa163e81bcc3  e47df157-e654-4491-a25d-ad42475d4822  i01p015-kube-node04     in-use
logging/data-elastic-0                pvc-c627c780-93ec-11e8-9844-fa163e81bcc3  69237173-6413-450b-9007-ec3bce8b3e39  i01p015-kube-node03     in-use
````

## kubectl openstack lb

The `lb` command combines information about Kubernetes Services with OpenStack LoadBalancer resources.

````
$ kubectl openstack lb
NAME                  FLOATING_IPS  VIP_ADDRESS  PORTS                                            SERVICES
external              59.1.0.15     10.12.4.6    8080 => [10.12.4.17 10.12.4.7 10.12.4.15]:30080  external/traefik
internal              59.1.0.14     10.12.4.5    443 => [10.12.4.17 10.12.4.7 10.12.4.15]:30443   internal/traefik
````

# Roadmap

* enable output via go template like json path (from both openstack & kube object)
* unit tests
