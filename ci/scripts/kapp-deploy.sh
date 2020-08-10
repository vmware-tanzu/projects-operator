#!/usr/bin/env bash

export PATH="${PATH}:/root/go/bin"

if [ -n "$KUBECONFIG_FILE_CONTENTS" ]; then
    mkdir -p "${HOME}/.kube"
    echo "$KUBECONFIG_FILE_CONTENTS" > "${HOME}/.kube/config"
fi

mkdir -p "${HOME}/.docker"
DOCKER_AUTH=$(echo -n "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" | base64 - | tr -d '\n')
cat <<EOT > "${HOME}/.docker/config.json"
{
  "auths": {
    "${REGISTRY_HOSTNAME}/${REGISTRY_PROJECT}": {
        "auth": "${DOCKER_AUTH}"
    }
  }
}
EOT

VERSION=$(cat version/version)

cd projects-operator
CI=true make kapp-deploy
