# Projects

## About

`projects-operator` extends kubernetes with a `Project` CRD and corresponding
controller.  `Projects` are intended to provide isolation of kubernetes
resources on a single kubernetes cluster.  A `Project` is essentially a
kubernetes namespace along with a corresponding set of RBAC rules.

## Usage

`projects-operator` is currently deployed using [helm (v3)](https://helm.sh/).

You must first create a `ClusterRole` that contains the RBAC
rules you wish to apply to each of the `Project`s. For example:

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
apiVersion: projects.pivotal.io/v1alpha1
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

### Uninstalling via helm

```
helm uninstall projects-operator
```

Note that the `Project` CRD will be left on the cluster as will any CRs for the `Project` CRD. These can be deleted manually if desired.

### Webhooks

projects-operator makes use of three webhooks to provide further functionality, as follows:

1. A ValidatingWebhook (invoked on Project CREATE) - ensures that Projects cannot be created if they have the same name as an existing namespace.
1. A MutatingWebhook (invoked on ProjectAccess CREATE, UPDATE) - returns a modified ProjectAccess containing the list of Projects the user has access to.
1. A MutatingWebhook (invoked on Project CREATE) - adds the user from the request as a member of the project if a project is created with no entries in access.
