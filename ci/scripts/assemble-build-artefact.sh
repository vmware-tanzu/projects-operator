#!/bin/sh

mkdir archive
tar -czf archive/projects-$(cat version/number).tgz marketplace-project/config
