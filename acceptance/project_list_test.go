package acceptance_test

import (
	"fmt"
	"time"

	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProjectAccess CRD", func() {
	var (
		// Note that these users have a corresponding identity in LDAP
		// see the README.md for further info
		alana testhelpers.KubeActor // an "admin/operator"
		cody  testhelpers.KubeActor // a "developer"

		projectName       string
		projectAccessName string
		projectResource   string
	)

	BeforeEach(func() {
		alana = testhelpers.NewKubeDefaultActor()

		codyToken := GetToken(Params.UaaLocation, "cody", Params.CodyPassword)
		cody = testhelpers.NewKubeActor("cody", codyToken)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
		projectAccessName = fmt.Sprintf("my-projectaccess-%d", time.Now().UnixNano())

		projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, projectName, userName("cody"))

		alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
	})

	AfterEach(func() {
		alana.MustRunKubectl("delete", "projectaccess", projectAccessName)

		alana.MustRunKubectl("delete", "-f", AsFile(projectResource))
	})

	It("can be created", func() {
		projectAccessResource := fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: %s`, projectAccessName)

		_, err := alana.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())
	})

	It("eventually has its status updated to include a list of projects cody has access to", func() {
		projectAccessResource := fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: %s`, projectAccessName)

		_, err := cody.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			output, err := cody.RunKubeCtl("get", "projectaccess", projectAccessName, "-o", "yaml")
			if err != nil {
				fmt.Fprint(GinkgoWriter, err.Error())
			}
			return output
		}, time.Second*3).Should(ContainSubstring(projectName))
	})
})
