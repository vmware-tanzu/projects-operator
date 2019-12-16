# Projects

### Usage

```
make install
make run
```

Apply the projects yaml with a project name and a list of users/serviceaccounts who have access.
```
apiVersion: developerconsole.pivotal.io/v1
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

### Uninstalling

The helm chart can be uninstalled with:
```
kubectl delete -f config/crd/bases
```
Note that the `Project` CRD will be left on the cluster as will any CRs for the `Project` CRD. These can be deleted manually if desired.

##### Limitations

Right now projects is hardcoded for usage by Developer Console to restrict users to a set of ServiceCatalog resources. This will be removed in the future. 

In order to configure arbitary resources you must change the following configuration:
1. The controller environment vars for [role permissions](https://github.com/pivotal/projects-operator/blob/master/config/manager/manager.yaml#L40-L45).
1. The controller's [own permissions](https://github.com/pivotal/projects-operator/blob/master/controllers/project_controller.go#L54-L55) since the controller must have permission to resources it creates.


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

The following dependencies need to be installed in order to hack on the projects operator:

* [Go](https://golang.org/doc/install)
  * [ginkgo](https://github.com/onsi/ginkgo)
  * [counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) (v6)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kubebuilder 2.0](https://github.com/kubernetes-sigs/kubebuilder)


