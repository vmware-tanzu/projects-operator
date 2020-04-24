# Contributing

When contributing to this repository, please first discuss the change you wish to make via [GitHub issue](https://github.com/pivotal/projects-operator/issues) before making a pull request.

### Dependencies

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

### Development and testing workflow

We are  using [skaffold](https://github.com/GoogleContainerTools/skaffold) for
development and testing workflow. In order to use this workflow, you must first
download `skaffold` and then update the `skaffold.yaml` file as required.
Specifically you will need to point to an image registry you have access to and
to set the `clusterRoleRef`. The default `clusterRoleRef` is set to the name of
the role the acceptance tests use.

#### Running tests with a kubernetes cluster

1. Target a cluster using OIDC pointing to an LDAP.
    1. You can set up openldap as a container by running `./ldap/deploy-ldap.sh`
    1. Run `./ldap/generate-uesrs.sh <ldap_host> <admin_dn> <admin_password>` and take note of the generated password.
1. Ensure that a `registry-secret` exists for your registry in the namespace you are deploying to.
    1. `kubectl create secret docker-registry registry-secret --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-password> --docker-email=<your-email>`

In one terminal window:

```
$ export CLUSTER_ROLE_REF="my-clusterrole"
$ make install
$ make dev
```

This will first apply the necessary RBAC rules to the cluster and do initial setup,
then monitor for changes on the codebase and then build, tag and push a new
image to the configured registry. It will then also do a helm update. Once the
helm update has completed you are free to run your tests.

In a separate window:

1. Set up the following environment variables:
    `export UAA_LOCATION=<UAA_SERVER_LOCATION>`
    `export CLUSTER_API_LOCATION=<CLUSTER>`
    `export CLUSTER_NAME=<CLUSTER_NAME>`
    `export DEVELOPER_PASSWORD=<PASSWORD_ABOVE>`
    `export OIDC_USER_PREFIX=<OIDC_USER_PREFIX>` (optional)
    `export OIDC_GROUP_PREFIX=<OIDC_GROUP_PREFIX>` (optional)
1. Run `make test`

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

To run the controller locally using go:

```
$ export CLUSTER_ROLE_REF="my-clusterrole"
$ make install && make run
```