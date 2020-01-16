package acceptance_test

import (
	"fmt"
	"time"

	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = PDescribe("ProjectAccess CRD", func() {
	var (
		// Note that these users have a corresponding identity in LDAP
		// see the README.md for further info
		alana testhelpers.KubeActor // an "admin/operator"
		cody  testhelpers.KubeActor // a "developer"
	)

	BeforeEach(func() {
		alana = testhelpers.NewKubeDefaultActor()

		codyToken := GetToken(Params.UaaLocation, "cody", Params.CodyPassword)
		cody = testhelpers.NewKubeActor("cody", codyToken)

		// grant cody ability to crud projectaccesses
		projectAccessClusterRole := `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: "cody-projectaccess"
rules:
- apiGroups:
  - developerconsole.pivotal.io
  resources:
  - projectaccesses
  verbs:
  - "*"
`

		projectAccessClusterRoleBinding := `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: "cody-projectaccess"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cody-projectaccess
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: cody
`

		// TODO: clean this up
		alana.MustRunKubectl("apply", "-f", AsFile(projectAccessClusterRole))
		alana.MustRunKubectl("apply", "-f", AsFile(projectAccessClusterRoleBinding))
	})

	AfterEach(func() {
		alana.MustRunKubectl("delete", "projectaccess", "cody-project-access")
	})

	It("can be created", func() {
		projectAccessResource := `
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: cody-project-access`

		_, err := alana.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())
	})

	It("eventually has its status updated to include a list of projects cody has access to", func() {
		projectAccessResource := `
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: cody-project-access`

		_, err := cody.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())

		//TODO: greatly improve this part of the test
		Eventually(func() string {
			output, err := cody.RunKubeCtl("get", "projectaccess", "cody-project-access", "-o", "yaml")
			if err != nil {
				fmt.Fprint(GinkgoWriter, err.Error())
			}

			return output
		}, time.Second*3).Should(ContainSubstring("ldap-experts"))
	})
})
