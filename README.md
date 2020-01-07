# Projects

## About

`projects-operator` extends kubernetes with a `Project` CRD and corresponding controller.
`Projects` are intended to provide isolation of kubernetes resources on a single kubernetes cluster.
A `Project` is essentially a kubernetes namespace along with a corresponding set of RBAC rules.

## Usage

`projects-operator` is currently deployed using [helm (v3+)](https://helm.sh/), but it is also possible to run the controller locally for development and testing purposes.

NB: In both cases you must first create a `ClusterRole` that contains the RBAC rules you wish to apply to each of the `Project`s. For example:

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

### Running locally

To run the controller locally using go:

```
$ export CLUSTER_ROLE_REF="my-clusterrole"
$ make install && make run
```

### Deploying via helm

```
# 1. Build the controller manager image

$ docker build -t my-registry/projects-operator:my-tag .

# 2. Push the controller manager image

$ docker push my-registry/projects-operator:my-tag

# 3. Helm deploy, setting the clusterRoleRef and the image

$ helm install projects-operator -helm/projects-operator \
  --set clusterRoleRef=my-clusterrole  \
  --set image=my-registry/projects-operator:my-tag
```

### Creating a Project

Apply projects yaml with a project name and a list of users/groups/serviceaccounts who have access, for example:

```
apiVersion: developerconsole.pivotal.io/v1alpha1
kind: Project
metadata:
  name: project-sample
spec:
  access:
  - kind: User
    name: cody
  - kind: ServiceAccount
    name: some-robot
    namespace: some-namespace
  - kind: Group
    name: ldap-experts
```

### Uninstalling via helm

```
helm uninstall projects-operator
```

Note that the `Project` CRD will be left on the cluster as will any CRs for the `Project` CRD. These can be deleted manually if desired.

### Tests

To run the acceptance tests you must have a pks k8s cluster using OIDC pointing to an LDAP. (You can set up openldap as a container by running `./ldap/deploy-ldap.sh`
1. Run `./ldap/generate-users.sh <ldap_host> <admin_dn> <admin_password>` and take note of the generated password
1. Setup the following env vars: 
  1. `export UAA_LOCATION=<UAA_SERVER_LOCATION>`
  1. `export CLUSTER_API_LOCATION=<CLUSTER>`
  1. `export CLUSTER_NAME=<CLUSTER_NAME>`
  1. `export CODY_PASSWORD=<PASSWORD_ABOVE>`
  1. `export OIDC_USER_PREFIX=<OIDC_USER_PREFIX>` (optional)
  1. `export OIDC_GROUP_PREFIX=<OIDC_GROUP_PREFIX>` (optional)

### Development

The following dependencies need to be installed in order to hack on projects-operator:

* [Go](https://golang.org/doc/install)
  * [ginkgo](https://github.com/onsi/ginkgo)
  * [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) (v6)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kubebuilder 2.0](https://github.com/kubernetes-sigs/kubebuilder)
* [golangci-lint](https://github.com/golangci/golangci-lint)
* [docker](https://www.docker.com/)
* [helm](https://helm.sh/) (v3)
