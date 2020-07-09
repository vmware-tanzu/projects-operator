#!/bin/sh

VERSION=$(cat version/version)

sed -i "s/version:.*/version:\ ${VERSION}/" projects-operator/deployments/k8s/values.yaml

tar -czf projects-operator-${VERSION}.tgz -C projects-operator/deployments -s /k8s/projects-operator/ k8s/
mv projects-operator-${VERSION}.tgz archive/projects-operator-${VERSION}.tgz