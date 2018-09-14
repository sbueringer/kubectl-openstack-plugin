#!/usr/bin/env bash

# Kubectl 1.12
sudo cp ./run.sh /usr/local/bin/kubectl-os

#go build cmd/kubectl-os.go
#sudo mv kubectl-os /usr/local/bin/kubectl-os

# Kubectl < 1.12
#PLUGIN_FOLDER="/home/fedora/.kube/plugins/os"
#rm -rf ${PLUGIN_FOLDER}
#mkdir -p ${PLUGIN_FOLDER}
#cp ./run.sh ${PLUGIN_FOLDER}/run.sh
#cp ./plugin.yaml ${PLUGIN_FOLDER}/plugin.yaml
#
#ls -la ${PLUGIN_FOLDER}


