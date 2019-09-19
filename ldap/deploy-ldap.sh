#!/usr/bin/env bash
set -eu

NAMESPACE=openldap
ADMIN_SECRET_NAME=openldap-admin-secret

kubectl create ns $NAMESPACE

absolute_path() {
  (cd "$1" && pwd)
}
scripts_path=$(absolute_path "$(dirname "$0")" )

set +e
kubectl -n $NAMESPACE get secret $ADMIN_SECRET_NAME 1>/dev/null 2>&1
secret_exists=$?
set -e

if [ $secret_exists -eq 0 ]
then
  echo "$ADMIN_SECRET_NAME secret already exists, reusing"
  adminPassword="$(kubectl -n $NAMESPACE get secret $ADMIN_SECRET_NAME -o json | jq -r '.data.password' | base64 --decode)"
else
  echo "$ADMIN_SECRET_NAME Secret not found, creating"
  adminPassword="$(cat /dev/urandom | env LC_CTYPE=C tr -dc a-z | head -c 32)"

cat <<EOF | kubectl -n $NAMESPACE create -f -
apiVersion: v1
kind: Secret
metadata:
  name: $ADMIN_SECRET_NAME
type: Opaque
data:
  password: "$(echo $adminPassword | base64)"
EOF
fi

kubectl -n $NAMESPACE apply -f ${scripts_path}/storage-class-gcp.yml
kubectl -n $NAMESPACE apply -f ${scripts_path}/open-ldap-deployment.yaml
kubectl -n $NAMESPACE apply -f ${scripts_path}/php-ldap-admin-deployment.yaml

echo
echo "Admin Credentials:"
echo "Login DN: cn=admin,dc=example,dc=com"
echo "Password: $adminPassword"

echo
echo "Once the service has a external IP assigned, to populate data, you can run:"
echo "    ${scripts_path}/generate-users.sh <external-ip> cn=admin,dc=example,dc=com $adminPassword"
