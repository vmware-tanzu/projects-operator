#!/bin/sh

export PATH="$PATH:/root/go/bin"

if [ -n "$KUBECONFIG_FILE_CONTENTS" ]; then
    mkdir -p "$HOME/.kube"
    echo "$KUBECONFIG_FILE_CONTENTS" > "$HOME/.kube/config"
fi

mkdir -p $HOME/.docker
DOCKER_AUTH=$(echo -n "$REGISTRY_USERNAME:$REGISTRY_PASSWORD" | base64 - | tr -d '\n')
cat <<EOT > $HOME/.docker/config.json
{
  "auths": {
    "${REGISTRY_URL}": {
        "auth": "${DOCKER_AUTH}"
    }
  }
}
EOT

# install kapp (TODO: put in the base ci image)
wget "https://github.com/k14s/kapp/releases/download/v0.30.0/kapp-linux-amd64" -O "/usr/local/bin/kapp"
chmod u+x "/usr/local/bin/kapp"

# install kbld (TODO: put in the base ci image)
wget "https://github.com/k14s/kbld/releases/download/v0.23.0/kbld-linux-amd64" -O "/usr/local/bin/kbld"
chmod u+x "/usr/local/bin/kbld"

VERSION=$(cat version/version)

cd projects-operator
CI=true make kapp-deploy
