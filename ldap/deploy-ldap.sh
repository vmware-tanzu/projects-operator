#!/usr/bin/env bash

kubectl get secret openldap-admin-secret 1>/dev/null 2>&1
secret_exists=$?

set -eu


if [ $secret_exists -eq 0 ]
then
  echo "openldap-admin-secret Secret already exists, reusing"
  adminPassword="$(kubectl get secret openldap-admin-secret -o json | jq -r '.data.password' | base64 --decode)"
else
  echo "openldap-admin-secret Secret not found, creating"
  adminPassword="$(cat /dev/urandom | env LC_CTYPE=C tr -dc a-z | head -c 32)"

cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Secret
metadata:
  name: openldap-admin-secret
type: Opaque
data:
  password: "$(echo $adminPassword | base64)"
EOF

fi


kubectl apply -f storage-class-gcp.yml

kubectl apply -f open-ldap-deployment.yaml
kubectl apply -f php-ldap-admin-deployment.yaml


echo
echo "Admin Credentials:"
echo "Login DN: cn=admin,dc=example,dc=com"
echo "Password: $adminPassword"

echo
echo "Once the service has a external IP assigned, to populate data, you can run:"
echo "    ./generate-users.sh <external-ip> cn=admin,dc=example,dc=com $adminPassword"