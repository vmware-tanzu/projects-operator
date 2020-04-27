# Contributing

When contributing to this repository, please first discuss the change you wish to make via [GitHub issue](https://github.com/pivotal/projects-operator/issues) before making a pull request.

## Dependencies

The following dependencies need to be installed in order to hack on projects-operator:

* [Go](https://golang.org/doc/install)
  * [ginkgo](https://github.com/onsi/ginkgo)
  * [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) (v6)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kubebuilder 2.0](https://github.com/kubernetes-sigs/kubebuilder)
* [golangci-lint](https://github.com/golangci/golangci-lint) (v1.24.0+)
* [docker](https://www.docker.com/)
* [helm](https://helm.sh/) (v3)
* [skaffold](https://github.com/GoogleContainerTools/skaffold) (v1.6.0+)

## Testing your contribution

### Unit tests

Unit tests can be run with:
```
$ make unit-tests
```

### Acceptance tests

Acceptance tests are run against an installation of projects-operator on a cluster using OIDC backed by LDAP, there are a number
of ways to install projects-operator however the recommended way for development is to use Skaffold.

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

#### Installing projects-operator via skaffold

In order to use this workflow, you must first download [skaffold](https://github.com/GoogleContainerTools/skaffold)
and then update the `skaffold.yaml` file as required. Specifically you will need to point to an image registry you
have access to and to set the `clusterRoleRef`. The default `clusterRoleRef` is set to the name ofthe role the
acceptance tests use. N.B. `skaffold` must be v1.6.0+.

You also need to ensure that a `docker-registry` secret named `registry-secret` exists for your registry in the
namespace you are deploying to, this can be done by running:
```
$ kubectl create secret docker-registry registry-secret --docker-server=<your-registry-server> --docker-username=<your-username> --docker-password=<your-password> --docker-email=<your-email>
```

Once configured, in a new terminal window you can then run:
```
$ make dev
```
This will monitor for changes on the codebase and then build, tag and push a new image to the configured registry.
It will then also do a helm update. Once the helm update has completed you are free to run your tests.

#### Running the tests

Setup the following env vars:
```
export UAA_LOCATION=<UAA_SERVER_LOCATION>
export CLUSTER_API_LOCATION=<CLUSTER>
export CLUSTER_NAME=<CLUSTER_NAME>
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
