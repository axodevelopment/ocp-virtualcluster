# ocp-virtualcluster

Prototype of a virtualcluster

#Process

operator-sdk init --domain prototypes.com --repo github.com/axodevelopment/ocp-virtualcluster/controller

operator-sdk create api --group organization --version v1 --kind VirtualCluster --resource --controller

go get kubevirt.io/client-go/kubecli@latest

edited api/vq vc_types.go

make docker-build docker-push IMG=docker.io/axodevelopment/src:latest
