# Projects

## About

`projects-operator` extends kubernetes with a `Project` CRD and corresponding
controller.  `Projects` are intended to provide isolation of kubernetes
resources on a single kubernetes cluster.  A `Project` is essentially a
kubernetes namespace along with a corresponding set of RBAC rules.

## Contributing

To begin contributing, please read the [contributing](CONTRIBUTING.md) doc.

## Installation and Usage

`projects-operator` is currently deployed using [k14s](https://k14s.io).

You must first create a `ClusterRole` that contains the RBAC
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

### Install

Then you will need to build and push the projects-operator image to a registry.

```bash
$ docker build -t <REGISTRY_HOSTNAME>/<REGISTRY_PROJECT>/projects-operator .
$ docker push <REGISTRY_HOSTNAME>/<REGISTRY_PROJECT>/projects-operator

# For example, docker build -t gcr.io/team-a/projects-operator .
```

Then finally you can run the [/scripts/kapp-deploy](/scripts/kapp-deploy) script
to deploy projects-operator.

```bash

export INSTANCE=<UNIQUE STRING TO IDENTIFY THIS DEPLOYMENT>
export REGISTRY_HOSTNAME=<REGISTRY_HOSTNAME> # e.g. "gcr.io", "my.private.harbor.com", etc.
export REGISTRY_PROJECT=<REGISTRY_PROJECT>   # e.g. "team-a", "dev", etc.
export REGISTRY_USERNAME=<REGISTRY_PASSWORD>
export REGISTRY_PASSWORD=<REGISTRY_PASSWORD>
export CLUSTER_ROLE_REF=my-clusterrole-with-rbac-for-each-project

$ ./scripts/kapp-deploy
```

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
kapp -n <NAMESPACE> delete -a projects-operator
```

### Webhooks

projects-operator makes use of three webhooks to provide further functionality, as follows:

1. A ValidatingWebhook (invoked on Project CREATE) - ensures that Projects cannot be created if they have the same name as an existing namespace.
1. A MutatingWebhook (invoked on ProjectAccess CREATE, UPDATE) - returns a modified ProjectAccess containing the list of Projects the user has access to.
1. A MutatingWebhook (invoked on Project CREATE) - adds the user from the request as a member of the project if a project is created with no entries in access.
