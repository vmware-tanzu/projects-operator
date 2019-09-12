#!/bin/sh

mkdir archive
tar -czf archive/projects.tgz \
  marketplace-project/config/crd/bases/marketplace.pivotal.io_projects.yaml \
  marketplace-project/deployment/deployment.yaml
