#!/usr/bin/env bash

set -eu

host=$1
dn=$2
password=$3

NAMESPACE=openldap

set +e
kubectl -n $NAMESPACE get secret openldap-user-secret 1>/dev/null 2>&1
secret_exists=$?
set -e

if [ $secret_exists -eq 0 ]
then
  echo "openldap-user-secret Secret already exists, reusing"
  userPassword="$(kubectl -n $NAMESPACE get secret openldap-user-secret -o json | jq -r '.data.password' | base64 --decode)"
else
  echo "openldap-user-secret Secret not found, creating"
  userPassword="$(cat /dev/urandom | env LC_CTYPE=C tr -dc a-z | head -c 32 | base64)"

cat <<-EOF | kubectl -n $NAMESPACE apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: openldap-user-secret
type: Opaque
data:
  password: $userPassword
EOF
fi

hashedPassword="$(slappasswd -s $userPassword -h {SSHA})"

ldapadd -c -h $host -D $dn -w $password <<EOF

dn: cn=cody,dc=example,dc=com
cn: cody
givenname: Enterprise Resource
mail: cody@example.com
objectclass: inetOrgPerson
objectclass: top
sn: Unit
userpassword: $hashedPassword

dn: cn=alice,dc=example,dc=com
cn: alice
givenname: Enterprise Resource
mail: alice@example.com
objectclass: inetOrgPerson
objectclass: top
sn: Unit
userpassword: $hashedPassword

dn: cn=bob,dc=example,dc=com
cn: bob
givenname: Enterprise Resource
mail: bob@example.com
objectclass: inetOrgPerson
objectclass: top
sn: Unit
userpassword: $hashedPassword

dn: cn=ldap-experts,dc=example,dc=com
cn: ldap-experts
member: cn=cody,dc=example,dc=com
objectclass: groupOfNames
objectclass: top
EOF

echo
echo "Users created:"
echo "user: user password: $userPassword"
