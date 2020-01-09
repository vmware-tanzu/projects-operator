package acceptance_test

import (
	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProjectAccess CRD", func() {
	var (
		// Note that these users have a corresponding identity in LDAP
		// see the README.md for further info
		alana testhelpers.KubeActor // an "admin/operator"
	)

	BeforeEach(func() {
		alana = testhelpers.NewKubeDefaultActor()
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
})
