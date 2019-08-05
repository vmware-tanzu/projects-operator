package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	marketplacev1 "github.com/pivotal-cf/marketplace-project/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project", func() {
	var (
		project        *marketplacev1.Project
		projectName    string
		tmpProjectFile *os.File
	)

	BeforeEach(func() {
		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())

		project = &marketplacev1.Project{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Project",
				APIVersion: "marketplace.pivotal.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: projectName,
			},
		}
	})

	JustBeforeEach(func() {
		var err error
		tmpProjectFile, err = ioutil.TempFile("", "project")
		Expect(err).NotTo(HaveOccurred())

		projectContent, err := json.Marshal(project)
		Expect(err).NotTo(HaveOccurred())

		_, err = tmpProjectFile.Write(projectContent)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.Remove(tmpProjectFile.Name())).To(Succeed())
	})

	When("a Project resource is created", func() {
		AfterEach(func() {
			RunKubectl("delete", "--wait=true", "-f", tmpProjectFile.Name())
		})

		It("creates a namespace with the project name", func() {
			RunKubectl("apply", "-f", tmpProjectFile.Name())

			Eventually(func() string {
				output, _ := exec.Command("kubectl", "get", "namespace", projectName).CombinedOutput()
				return string(output)
			}).Should(
				SatisfyAll(MatchRegexp("NAME\\s+STATUS\\s+"), MatchRegexp(fmt.Sprintf("%s\\s+Active\\s+", projectName))))
		})
	})

	When("a Project resource is deleted", func() {
		JustBeforeEach(func() {
			RunKubectl("apply", "-f", tmpProjectFile.Name())

			Eventually(func() string {
				output, _ := exec.Command("kubectl", "get", "namespace", projectName).CombinedOutput()
				return string(output)
			}).Should(SatisfyAll(MatchRegexp("NAME\\s+STATUS\\s+"), MatchRegexp(fmt.Sprintf("%s\\s+Active\\s+", projectName))))
		})

		It("deletes its corresponding namespace", func() {
			RunKubectl("delete", "-f", tmpProjectFile.Name())

			Eventually(func() string {
				output, _ := exec.Command("kubectl", "get", "namespace", projectName).CombinedOutput()
				return string(output)
			}).Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})
	})
})
