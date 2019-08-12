package acceptance_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project", func() {
	var (
		projectResource string
		projectName     string
		tmpProjectFile  *os.File
		token           string
		kubectl         Kubectl
	)

	BeforeEach(func() {

		token = GetToken(Params.UaaLocation, "eunit", Params.CodyPassword)
		kubectl = GetKubectlFor(Params.ClusterLocation, "marketplace-project-ci", token)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())

		projectResource = fmt.Sprintf(`---
apiVersion: marketplace.pivotal.io/v1
kind: Project
metadata:
  name: %s
`, projectName)

	})

	JustBeforeEach(func() {
		var err error
		tmpProjectFile, err = ioutil.TempFile("", "project")
		Expect(err).NotTo(HaveOccurred())

		_, err = tmpProjectFile.Write([]byte(projectResource))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Remove(tmpProjectFile.Name())).To(Succeed())
	})

	When("a Project resource is created", func() {
		AfterEach(func() {
			_, err := kubectl("delete", "--wait=true", "-f", tmpProjectFile.Name())
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates a namespace with the project name", func() {
			_, err := kubectl("apply", "-f", tmpProjectFile.Name())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				output, _ := kubectl("get", "namespace")
				return output
			}).Should(SatisfyAll(MatchRegexp("NAME\\s+STATUS\\s+"), MatchRegexp(fmt.Sprintf("%s\\s+Active\\s+", projectName))))
		})
	})

	When("a Project resource is deleted", func() {
		JustBeforeEach(func() {
			_, err := kubectl("apply", "-f", tmpProjectFile.Name())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				output, _ := kubectl("get", "namespace", projectName)
				return output
			}).Should(SatisfyAll(MatchRegexp("NAME\\s+STATUS\\s+"), MatchRegexp(fmt.Sprintf("%s\\s+Active\\s+", projectName))))
		})

		It("deletes its corresponding namespace", func() {
			_, err := kubectl("delete", "-f", tmpProjectFile.Name())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				output, _ := kubectl("get", "namespace", projectName)
				return output
			}).Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})
	})
})
