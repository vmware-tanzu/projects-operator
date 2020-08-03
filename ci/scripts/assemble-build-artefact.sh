#!/bin/sh

VERSION=$(cat version/version)

sed -i "s/version:.*/version:\ ${VERSION}/" projects-operator/deployments/k8s/values/_default.yaml

tar -czf projects-operator-${VERSION}.tgz -C projects-operator/deployments --transform "s/k8s/projects-operator/g" k8s/
mv projects-operator-${VERSION}.tgz archive/projects-operator-${VERSION}.tgz