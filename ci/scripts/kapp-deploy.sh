#!/bin/sh

export PATH="$PATH:/root/go/bin"

if [ -n "$KUBECONFIG_FILE_CONTENTS" ]; then
    mkdir -p "$HOME/.kube"
    echo "$KUBECONFIG_FILE_CONTENTS" > "$HOME/.kube/config"
fi

# install kapp (TODO: put in the base ci image)
wget "https://github.com/k14s/kapp/releases/download/v0.30.0/kapp-linux-amd64" -O "/usr/local/bin/kapp"
chmod u+x "/usr/local/bin/kapp"

VERSION=$(cat version/version)
PROJECTS_OPERATOR_IMAGE="${PROJECTS_OPERATOR_IMAGE}@$(cat project-operator-image/digest)"

cd projects-operator
CI=true make kapp-deploy
