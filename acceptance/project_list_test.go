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
		adminUser    testhelpers.KubeActor
		devUserAlice testhelpers.KubeActor

		projectName       string
		projectAccessName string
		projectResource   string
	)

	BeforeEach(func() {
		adminUser = testhelpers.NewKubeDefaultActor()

		devUserAliceToken := GetToken(Params.UaaLocation, "alice", Params.DeveloperPassword)
		devUserAlice = testhelpers.NewKubeActor("alice", devUserAliceToken)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
		projectAccessName = fmt.Sprintf("my-projectaccess-%d", time.Now().UnixNano())

		projectResource = fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, projectName, userName("alice"))

		adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
	})

	AfterEach(func() {
		adminUser.MustRunKubectl("delete", "projectaccess", projectAccessName)

		adminUser.MustRunKubectl("delete", "-f", AsFile(projectResource))
	})

	It("can be created", func() {
		projectAccessResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: %s`, projectAccessName)

		_, err := adminUser.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())
	})

	It("eventually has its status updated to include a list of projects alice has access to", func() {
		projectAccessResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: ProjectAccess
                metadata:
                  name: %s`, projectAccessName)

		_, err := devUserAlice.RunKubeCtl("apply", "-f", AsFile(projectAccessResource))
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			output, err := devUserAlice.RunKubeCtl("get", "projectaccess", projectAccessName, "-o", "yaml")
			if err != nil {
				fmt.Fprint(GinkgoWriter, err.Error())
			}
			return output
		}, time.Second*3).Should(ContainSubstring(projectName))
	})
})
