# Contributing

The projects-operator team welcomes contributions from the community. Before you start working with projects-operator, please read our [Developer Certificate of Origin]( https://cla.vmware.com/dco). All contributions to this repository must be signed as described on that page. Your signature certifies that you wrote the patch or have the right to pass it on as an open-source patch.

When contributing to this repository, please first discuss the change you wish to make via [GitHub issue](https://github.com/pivotal/projects-operator/issues) before making a pull request.

## Dependencies

The following dependencies need to be installed in order to hack on projects-operator:

* [Go](https://golang.org/doc/install)
  * [ginkgo](https://github.com/onsi/ginkgo)
  * [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) (v6+)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) (v2+)
* [golangci-lint](https://github.com/golangci/golangci-lint) (v1.24.0+)
* [docker](https://www.docker.com/)
* [k14s](https://k14s.io)

## Testing your contribution

### Unit tests

Unit tests can be run with:
```
$ make unit-tests
```

### Acceptance tests

Acceptance tests are run against an installation of projects-operator on a
cluster using OIDC backed by LDAP.  For notes on installing projects-operator,
please refer to the [/README.md](/READMD.md).

NB: You will need to ensure you have deployed projects-operator with a version
of the projects-operator image that contains your changes.

#### Configuring the LDAP server

You can set up openldap as a container by running:
```
$ ./ldap/deploy-ldap.sh
```
Test users can then be created by running:
```
$ ./ldap/generate-users.sh <ldap_host> <admin_dn> <admin_password>
```
Take note of the generated password.

This LDAP server must then be set up as the OIDC backing for the Kubernetes cluster to be used for acceptance, see
[OpenID Connect Tokens](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#openid-connect-tokens).

#### Running the tests

Setup the following env vars:
```
export UAA_LOCATION=<UAA_SERVER_LOCATION>
export DEVELOPER_PASSWORD=<DEVELOPER_PASSWORD>
export OIDC_USER_PREFIX=<OIDC_USER_PREFIX> (optional)
export OIDC_GROUP_PREFIX=<OIDC_GROUP_PREFIX> (optional)
```
Note: `DEVELOPER_PASSWORD` is password of the LDAP users generated in [Configuring LDAP server](#configuring-ldap-server).

Then run:
```
$ make acceptance-tests
```

#### Running locally

It is possible to run the controller locally for development and testing purposes, though the webhooks cannot be tested locally. To test webhook-related features you will need to deploy the webhook to a kubernetes cluster.

You must first create a ClusterRole that contains the RBAC rules you wish to apply to each of the Projects. For example:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: my-clusterrole
rules:
- apiGroups:
  - example.k8s.io
  resources:
  - mycustomresource
  verbs:
  - "*"
```

To run the controller locally using Go:

```
$ export CLUSTER_ROLE_REF="my-clusterrole"
$ make install && make run
```
