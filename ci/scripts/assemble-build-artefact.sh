#!/bin/sh


VERSION=$(cat version/number)
sed -i s/version:.*/version:\ ${VERSION}/ marketplace-project/helm/projects-operator/Chart.yaml
sed -i s/appVersion:.*/appVersion:\ ${VERSION}/ marketplace-project/helm/projects-operator/Chart.yaml
helm package marketplace-project/helm/projects-operator/ -d archive/
