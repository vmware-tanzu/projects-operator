#!/usr/bin/env bash

host=$1
dn=$2
password=$3

kubectl get secret openldap-user-secret 1>/dev/null 2>&1
secret_exists=$?

set -eu


if [ $secret_exists -eq 0 ]
then
  echo "openldap-user-secret Secret already exists, reusing"
  userPassword="$(kubectl get secret openldap-user-secret -o json | jq -r '.data.password' | base64 --decode)"
else
  echo "openldap-user-secret Secret not found, creating"
  userPassword="$(cat /dev/urandom | env LC_CTYPE=C tr -dc a-z | head -c 32 | base64)"

cat <<-EOF | kubectl apply -f -
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

dn: cn=posix-group,dc=example,dc=com
cn: posix-group
gidnumber: 500
objectclass: posixGroup
objectclass: top

dn: cn=cody,dc=example,dc=com
cn: cody
gidnumber: 500
givenname: Enterprise Resource
homedirectory: /home/users/cody
mail: cody@example.com
objectclass: inetOrgPerson
objectclass: posixAccount
objectclass: top
sn: Unit
uid: cody
uidnumber: 1000
userpassword: $hashedPassword

dn: cn=alice,dc=example,dc=com
cn: alice
gidnumber: 500
givenname: Enterprise Resource
homedirectory: /home/users/alice
mail: alice@example.com
objectclass: inetOrgPerson
objectclass: posixAccount
objectclass: top
sn: Unit
uid: alice
uidnumber: 1000
userpassword: $hashedPassword

dn: cn=bob,dc=example,dc=com
cn: bob
gidnumber: 500
givenname: Enterprise Resource
homedirectory: /home/users/bob
mail: bob@example.com
objectclass: inetOrgPerson
objectclass: posixAccount
objectclass: top
sn: Unit
uid: bob
uidnumber: 1000
userpassword: $hashedPassword

dn: cn=ldap-experts,dc=example,dc=com
cn: ldap-experts
member: cn=eunit,dc=example,dc=com
objectclass: groupOfNames
objectclass: top

EOF

echo
echo "Users created:"
echo "user: eunit password: $userPassword"