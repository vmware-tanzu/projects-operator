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
		alana           KubeContext
		alice           KubeContext
		cody            KubeContext
		bob             KubeContext
	)

	Describe("Creating a project and using it", func() {

		BeforeEach(func() {

			codyToken := GetToken(Params.UaaLocation, "cody", Params.CodyPassword)
			cody = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", codyToken)
			aliceToken := GetToken(Params.UaaLocation, "alice", Params.CodyPassword)
			alice = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", aliceToken)
			bobToken := GetToken(Params.UaaLocation, "bob", Params.CodyPassword)
			bob = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", bobToken)
			alana = GetContextForAlana()

			projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
			projectResource = fmt.Sprintf(`---
apiVersion: marketplace.pivotal.io/v1
kind: Project
metadata:
 name: %s
spec:
  access:
  - kind: User
    name: cody
  - kind: User
    name: alice
`, projectName)
		})

		It("Cody can not add resources by default", func() {
			message, err := cody.Kubectl("create", "configmap", "test-map")

			Expect(err).To(HaveOccurred())
			Expect(message).To(ContainSubstring(`"cody" cannot create resource "configmaps"`))
		})

		It("Cody can not create projects", func() {
			_, err := cody.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).To(HaveOccurred())
		})

		It("Alana can create a project and Cody and Alice can add a resource into it, Bob can not", func() {
			message, err := alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)

			Eventually(cody.TryKubectl("-n", projectName, "create", "configmap", "test-map-cody")).
				Should(ContainSubstring("created"))

			Eventually(alice.TryKubectl("-n", projectName, "create", "configmap", "test-map-alice")).
				Should(ContainSubstring("created"))

			message, err = bob.Kubectl("-n", projectName, "create", "configmap", "test-map-bob")
			Expect(err).To(HaveOccurred(), message)

			Eventually(cody.TryKubectl("-n", projectName, "get", "configmaps")).
				Should(SatisfyAll(
					ContainSubstring("test-map-cody"),
					ContainSubstring("test-map-alice"),
					Not(ContainSubstring("test-map-bob")),
				))
		})

		It("Alana can revoke access to a project from cody", func() {
			message, err := alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)

			Eventually(cody.TryKubectl("-n", projectName, "create", "configmap", fmt.Sprintf("configmap-%d", time.Now().UnixNano()))).
				Should(ContainSubstring("created"))

			projectResource = fmt.Sprintf(`---
apiVersion: marketplace.pivotal.io/v1
kind: Project
metadata:
 name: %s
spec:
  access:
  - kind: User
    name: alice
`, projectName)

			message, err = alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)

			Eventually(func() string {
				m, _ := cody.Kubectl("-n", projectName, "create", "configmap", fmt.Sprintf("configmap-%d", time.Now().UnixNano()))
				return m
			}).Should(ContainSubstring(`"cody" cannot create resource "configmaps"`))
		})

		It("Alana can delete a project", func() {
			_, err := alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(alana.TryKubectl("get", "namespace", projectName, "--output", "jsonpath={.status.phase}")).
				Should(Equal("Active"))

			_, err = alana.Kubectl("delete", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(alana.TryKubectl("get", "namespace", projectName)).
				Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})

	})
})
