#!/bin/bash

if [ $# -lt 2 ]; then
	echo "Usage:"
	echo "  $0 k8s-config-file-merging-from destination-config-file"
	exit 1
fi

KUBECONFIG=~/.kube/config:$1 kubectl config view --flatten > $2