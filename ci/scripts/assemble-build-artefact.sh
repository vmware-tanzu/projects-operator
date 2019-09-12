#!/bin/sh

mkdir archive
tar -czf archive/projects.tgz \
  -C marketplace-project/config/crd/bases \
  marketplace.pivotal.io_projects.yaml
