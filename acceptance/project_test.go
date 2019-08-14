package acceptance_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project Resources", func() {
	var (
		projectName string
		alana       KubeContext
		alice       KubeContext
		cody        KubeContext
	)

	BeforeEach(func() {
		alana = GetContextForAlana()

		aliceToken := GetToken(Params.UaaLocation, "alice", Params.CodyPassword)
		alice = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", aliceToken)
		codyToken := GetToken(Params.UaaLocation, "cody", Params.CodyPassword)
		cody = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", codyToken)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
	})

	It("Cody can not add resources by default", func() {
		message, err := cody.Kubectl("create", "configmap", "test-map")

		Expect(err).To(HaveOccurred())
		Expect(message).To(ContainSubstring(`"cody" cannot create resource "configmaps"`))
	})

	It("Cody can not create projects", func() {
		message, err := cody.Kubectl("create", "-f", AsFile(fmt.Sprintf(`
            apiVersion: marketplace.pivotal.io/v1
            kind: Project
            metadata:
              name: %s`, projectName)))

		Expect(err).To(HaveOccurred())
		Expect(message).To(ContainSubstring(`User "cody" cannot create resource "projects"`))
	})

	When("Alana creates a project for Users", func() {
		var projectResource string

		BeforeEach(func() {

			projectResource = fmt.Sprintf(`
                apiVersion: marketplace.pivotal.io/v1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: cody
                  - kind: User
                    name: alice`, projectName)

			message, err := alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)
		})

		It("Cody and Alice can add a resource into it", func() {

			Eventually(cody.TryKubectl("-n", projectName, "create", "configmap", "test-map-cody")).
				Should(ContainSubstring("created"))

			Eventually(alice.TryKubectl("-n", projectName, "create", "configmap", "test-map-alice")).
				Should(ContainSubstring("created"))

			Eventually(cody.TryKubectl("-n", projectName, "get", "configmaps")).
				Should(SatisfyAll(
					ContainSubstring("test-map-cody"),
					ContainSubstring("test-map-alice"),
				))
		})

		It("Alana can revoke access to a project from cody", func() {
			Eventually(cody.TryKubectl("-n", projectName, "create", "configmap", fmt.Sprintf("configmap-%d", time.Now().UnixNano()))).
				Should(ContainSubstring("created"))

			projectResource = fmt.Sprintf(`
                apiVersion: marketplace.pivotal.io/v1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: alice`, projectName)

			message, err := alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)

			Eventually(func() string {
				m, _ := cody.Kubectl("-n", projectName, "create", "configmap", fmt.Sprintf("configmap-%d", time.Now().UnixNano()))
				return m
			}).Should(ContainSubstring(`"cody" cannot create resource "configmaps"`))
		})

		It("Alana can delete a project", func() {
			Eventually(alana.TryKubectl("get", "namespace", projectName, "--output", "jsonpath={.status.phase}")).
				Should(Equal("Active"))

			_, err := alana.Kubectl("delete", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred())

			Eventually(alana.TryKubectl("get", "namespace", projectName)).
				Should(ContainSubstring(fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName)))
		})

	})

	When("Alana creates a project for ServiceAccounts created in a namespace", func() {

		var (
			projectResource  string
			accountNamespace string
			serviceAccount   KubeContext
		)
		BeforeEach(func() {

			accountNamespace = "users" + projectName

			message, err := alana.Kubectl("create", "namespace", accountNamespace)
			Expect(err).NotTo(HaveOccurred(), message)

			serviceAccountName := fmt.Sprintf("service-account-acceptance-test-%d", time.Now().UnixNano())
			token := CreateServiceAccount(alana, serviceAccountName, accountNamespace)

			serviceAccount = GetContextFor(Params.ClusterLocation, "marketplace-project-ci", token)

			projectResource = fmt.Sprintf(`
                apiVersion: marketplace.pivotal.io/v1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: ServiceAccount
                    name: %s
                    namespace: %s`, projectName, serviceAccountName, accountNamespace)

			message, err = alana.Kubectl("apply", "-f", AsFile(projectResource))
			Expect(err).NotTo(HaveOccurred(), message)
		})

		AfterEach(func() {
			message, err := alana.Kubectl("delete", "namespace", accountNamespace)
			Expect(err).NotTo(HaveOccurred(), message)
		})

		It("ServiceAccount can add a resource into it", func() {
			Eventually(serviceAccount.TryKubectl("-n", projectName, "create", "configmap", "test-map-serviceaccount")).
				Should(ContainSubstring("created"))
		})

	})

	It("does not allow unknown service types", func() {

		projectResource := `
                apiVersion: marketplace.pivotal.io/v1
                kind: Project
                metadata:
                 name: project
                spec:
                  access:
                  - kind: SomeUnknownKind
                    name: alice`

		message, err := alana.Kubectl("apply", "-f", AsFile(projectResource))

		Expect(err).To(HaveOccurred(), message)
		Expect(message).To(ContainSubstring("spec.access.kind in body should be one of [ServiceAccount User]"))

	})
})
