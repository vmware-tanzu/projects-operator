package acceptance_test

import (
	"fmt"
	"time"

	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Projects CRD", func() {
	var (
		projectName string

		// Note that these users have a corresponding identity in LDAP
		// see the README.md for further info
		alana testhelpers.KubeActor // an "admin/operator"
		alice testhelpers.KubeActor // a "developer"
		cody  testhelpers.KubeActor // a "developer"
	)

	BeforeEach(func() {
		alana = testhelpers.NewKubeDefaultActor()

		// TODO default namespace
		aliceToken := GetToken(Params.UaaLocation, "alice", Params.CodyPassword)
		alice = testhelpers.NewKubeActor("alice", aliceToken)

		codyToken := GetToken(Params.UaaLocation, "cody", Params.CodyPassword)
		cody = testhelpers.NewKubeActor("cody", codyToken)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
	})

	When("alice and cody have not been given access to a Project", func() {
		It("does not permit them to interact with allowed resources", func() {
			output, err := cody.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("cody"))))
		})
	})

	When("alice and cody have been given User access to a Project", func() {
		var projectResource string

		BeforeEach(func() {
			projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, projectName, userName("cody"), userName("alice"))

			alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			alana.MustRunKubectl("delete", "-f", AsFile(projectResource))

			Eventually(func() string {
				output, _ := alana.RunKubeCtl("get", "namespace", projectName)
				return output
			}).Should(
				ContainSubstring(
					fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
				))
		})

		It("permits alice and cody to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := alice.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-alice")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "create", "serviceaccount", "test-sa-cody")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := alice.RunKubeCtl("-n", projectName, "get", "configmaps")
				return output
			}).Should(SatisfyAll(
				ContainSubstring("test-map-alice"),
			))
		})

		It("does not permit alice or cody to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := alice.RunKubeCtl("-n", projectName, "create", "quota", "test-quota-alice")
				return output
			}).Should(ContainSubstring("forbidden"))

			Consistently(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		When("alana revokes access to a project from cody", func() {
			BeforeEach(func() {
				projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, projectName, userName("alice"))

				alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
			})

			It("prevents cody from interacting with the allowed resources", func() {
				Eventually(func() string {
					output, _ := cody.RunKubeCtl("-n", projectName, "get", "configmaps")
					return output
				}).Should(ContainSubstring(fmt.Sprintf(`User "%s" cannot list resource "configmaps"`, userName("cody"))))
			})
		})
	})

	When("the ldap-experts group has been given Group access to a Project", func() {
		var projectResource string

		BeforeEach(func() {
			projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: Group
                    name: %s`, projectName, groupName("ldap-experts"))

			alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			alana.MustRunKubectl("delete", "-f", AsFile(projectResource))

			Eventually(func() string {
				output, _ := alana.RunKubeCtl("get", "namespace", projectName)
				return output
			}).Should(
				ContainSubstring(
					fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
				))
		})

		It("permits members of the Group to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-cody")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "get", "configmaps,serviceaccounts")
				return output
			}).Should(SatisfyAll(
				ContainSubstring("test-map-cody"),
			))
		})

		It("does not permit members of the Group to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "create", "quota", "test-quota-quota")
				return output
			}).Should(ContainSubstring("forbidden"))

			Consistently(func() string {
				output, _ := cody.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		It("does not permit non-members of the Group to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := alice.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		When("alana revokes access to a project from the ldap-experts group", func() {
			BeforeEach(func() {
				projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access: []`, projectName)

				alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
			})

			It("prevents members of the group from interacting with the allowed resources", func() {
				Eventually(func() string {
					output, _ := cody.RunKubeCtl("-n", projectName, "get", "configmaps")
					return output
				}).Should(ContainSubstring(fmt.Sprintf(`User "%s" cannot list resource "configmaps"`, userName("cody"))))
			})
		})
	})

	When("a ServiceAccount has been given access to a Project", func() {
		var (
			projectResource string
			saNamespace     string
			sa              testhelpers.KubeActor
		)

		BeforeEach(func() {
			saNamespace = "users" + projectName
			alana.MustRunKubectl("create", "namespace", saNamespace)

			saName := fmt.Sprintf("service-account-acceptance-test-%d", time.Now().UnixNano())
			saToken := testhelpers.CreateServiceAccount(saName, saNamespace)

			sa = testhelpers.NewKubeActor(saName, saToken)

			projectResource = fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: ServiceAccount
                    name: %s
                    namespace: %s`, projectName, saName, saNamespace)

			alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			alana.MustRunKubectl("delete", "namespace", saNamespace)
			alana.MustRunKubectl("delete", "project", projectName)

			Eventually(func() string {
				output, _ := alana.RunKubeCtl("get", "namespace", projectName)
				return output
			}).Should(
				ContainSubstring(
					fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
				))
		})

		It("permits the ServiceAccount to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := sa.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-sa")
				return output
			}).Should(ContainSubstring("created"))
		})
	})

	When("Alana creates and then deletes the project", func() {
		BeforeEach(func() {
			projectResource := fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, projectName, userName("cody"), userName("alice"))

			alana.MustRunKubectl("apply", "-f", AsFile(projectResource))
			alana.MustRunKubectl("delete", "-f", AsFile(projectResource))
		})

		It("removes the corresponding namespace and leaves Cody and Alice unable to add resources", func() {
			Eventually(func() string {
				output, _ := alana.RunKubeCtl("get", "namespace", projectName)
				return output
			}).Should(ContainSubstring(
				fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
			))

			message, err := cody.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(message).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("cody"))))

			message, err = alice.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(message).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("alice"))))
		})
	})

	It("does not allow unknown access types", func() {
		projectResource := fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: project
                spec:
                  access:
                  - kind: SomeUnknownKind
                    name: %s`, userName("alice"))

		message, err := alana.RunKubeCtl("apply", "-f", AsFile(projectResource))
		Expect(err).To(HaveOccurred(), message)
		Expect(message).To(ContainSubstring("spec.access.kind in body should be one of [ServiceAccount User Group]"))
	})

	It("does not allow alice or cody to create projects", func() {
		projectResource := fmt.Sprintf(`
                apiVersion: developerconsole.pivotal.io/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, "project", userName("cody"), userName("alice"))

		output, err := cody.RunKubeCtl("create", "-f", AsFile(projectResource))
		Expect(err).To(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "projects"`, userName("cody"))))
	})
})

func userName(user string) string {
	if Params.OIDCUserPrefix == "" {
		return user
	}
	return Params.OIDCUserPrefix + ":" + user
}

func groupName(group string) string {
	if Params.OIDCGroupPrefix == "" {
		return group
	}
	return Params.OIDCGroupPrefix + ":" + group
}
