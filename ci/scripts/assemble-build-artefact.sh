#!/bin/sh

mkdir archive
tar -czf archive/projects-$(cat version/number).tgz -C marketplace-project config
