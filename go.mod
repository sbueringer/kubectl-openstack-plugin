module github.com/sbueringer/kubectl-openstack-plugin

require (
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.4.0
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20180912035003-be2c049b30cc
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20191005115622-2e41325d9e4b
	k8s.io/apimachinery v0.0.0-20191005115455-e71eb83a557c
	k8s.io/cli-runtime v0.0.0-20191005121332-4d28aef60981
	k8s.io/client-go v0.0.0-20191005115821-b1fd78950135
	k8s.io/utils v0.0.0-20190923111123-69764acb6e8e // indirect
)

replace github.com/gophercloud/gophercloud v0.4.0 => github.com/sbueringer/gophercloud v0.4.0

go 1.13
