# Projects

## About

`projects-operator` extends kubernetes with a `Project` CRD and corresponding
controller.  `Projects` are intended to provide isolation of kubernetes
resources on a single kubernetes cluster.  A `Project` is essentially a
kubernetes namespace along with a corresponding set of RBAC rules.

## Installation

### Prerequisites

* **kubectl**: For installation instructions, see [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) in the Kubernetes documentation.
* **Docker CLI**: For installation instructions, see the [Docker documentation](https://docs.docker.com/install).
* **k14s tools**: For installation instructions, see [k14s.io](https://k14s.io).
* **A Kubernetes cluster**: This will be where the CF Service Bridge components get installed.

You must also create a `ClusterRole` that contains the RBAC
rules you wish to be applied to each created `Project`. For example:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: my-clusterrole-with-rbac-for-each-project
rules:
- apiGroups:
  - example.k8s.io
  resources:
  - mycustomresource
  verbs:
  - "*"
```
The env var `CLUSTE_ROLE_REF` should be set to the name of the ClusterRole. 

### Standard install

The default image registry for cf-service-bridge is dev.registry.pivotal.io. If
you have access to this registry, then you need only set the `INSTANCE`, `REGISTRY_USERNAME`,
`REGISTRY_PASSWORD` and `CLUSTER_ROLE_REF` env vars, then run `./scripts/kapp-deploy`.

### Custom install

If you don't have access to the default registry, or if you are working on
cf-service-bridge and wish to deploy your local changes (i.e. for testing), we
will need to build and push the image to a custom registry. This can be done by
setting the following env vars:

```bash
export INSTANCE=<UNIQUE STRING TO IDENTIFY THIS DEPLOYMENT>
export REGISTRY_HOSTNAME=<REGISTRY_HOSTNAME> # e.g. "gcr.io", "my.private.harbor.com", etc.
export REGISTRY_PROJECT=<REGISTRY_PROJECT>   # e.g. "team-a", "dev", etc.
export REGISTRY_USERNAME=<REGISTRY_PASSWORD>
export REGISTRY_PASSWORD=<REGISTRY_PASSWORD>
export CLUSTER_ROLE_REF=my-clusterrole-with-rbac-for-each-project
```

Then run `make kapp-local-deploy`. NB: you will need to
have a docker daemon running and you will need to have run `docker login` for
the registry you are using.

## Using Projects Operator

### Creating a Project

Apply projects yaml with a project name and a list of users/groups/serviceaccounts who have access, for example:

```yaml
apiVersion: projects.vmware.com/v1alpha1
kind: Project
metadata:
  name: project-sample
spec:
  access:
  - kind: User
    name: alice
  - kind: ServiceAccount
    name: some-robot
    namespace: some-namespace
  - kind: Group
    name: ldap-experts
```

### Uninstall

```bash
make kapp-delete
```

### Webhooks

projects-operator makes use of three webhooks to provide further functionality, as follows:

1. A ValidatingWebhook (invoked on Project CREATE) - ensures that Projects cannot be created if they have the same name as an existing namespace.
1. A MutatingWebhook (invoked on ProjectAccess CREATE, UPDATE) - returns a modified ProjectAccess containing the list of Projects the user has access to.
1. A MutatingWebhook (invoked on Project CREATE) - adds the user from the request as a member of the project if a project is created with no entries in access.
