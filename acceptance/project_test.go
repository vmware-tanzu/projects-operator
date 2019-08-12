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
		token           string
		kubectl         Kubectl
	)

	Context("a non admin user", func() {

		BeforeEach(func() {

			token = GetToken(Params.UaaLocation, "eunit", Params.CodyPassword)
			kubectl = GetKubectlFor(Params.ClusterLocation, "marketplace-project-ci", token)

			projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())

			projectResource = fmt.Sprintf(`---
apiVersion: marketplace.pivotal.io/v1
kind: Project
metadata:
  name: %s`, projectName)
		})

		It("can create and delete a project", func() {
			_, err := kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				output, _ := kubectl("get", "namespace")
				return output
			}).Should(SatisfyAll(MatchRegexp("NAME\\s+STATUS\\s+"), MatchRegexp(fmt.Sprintf("%s\\s+Active\\s+", projectName))))

			_, err = kubectl("delete", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				output, _ := kubectl("get", "namespace", projectName)
				return output
			}).Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})
	})
})
