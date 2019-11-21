#!/bin/sh

VERSION=$(cat version/number)

sed -i s/version:.*/version:\ ${VERSION}/ projects-operator/helm/projects-operator/Chart.yaml
sed -i s/appVersion:.*/appVersion:\ ${VERSION}/ projects-operator/helm/projects-operator/Chart.yaml
sed -i s/^image:.*/image:\ gcr.io\\/cf-ism-0\\/projects-operator:${VERSION}/ projects-operator/helm/projects-operator/values.yaml

helm package projects-operator/helm/projects-operator/ -d archive/
