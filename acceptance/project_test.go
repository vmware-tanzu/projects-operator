package acceptance_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project Resources", func() {
	var (
		projectResource string
		projectName     string
		eunit           KubeContext
	)

	Context("as cody", func() {

		BeforeEach(func() {

			eunitToken := GetToken(Params.UaaLocation, "eunit", Params.CodyPassword)

			eunit = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", eunitToken)

			projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())

			projectResource = fmt.Sprintf(`---
apiVersion: marketplace.pivotal.io/v1
kind: Project
metadata:
  name: %s`, projectName)
		})

		It("can create and delete a project", func() {
			_, err := eunit.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(eunit.TryKubectl("get", "namespace", projectName, "--output", "jsonpath={.status.phase}")).
				Should(Equal("Active"))

			_, err = eunit.Kubectl("delete", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(eunit.TryKubectl("get", "namespace", projectName)).
				Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})

	})
})
