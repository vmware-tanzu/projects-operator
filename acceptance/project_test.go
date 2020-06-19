package acceptance_test

import (
	"encoding/json"
	"fmt"
	"time"

	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Projects CRD", func() {
	var (
		projectName      string
		otherProjectName string

		// Note that these users have a corresponding identity in LDAP
		// see the README.md for further info
		adminUser    testhelpers.KubeActor
		devUserAlice testhelpers.KubeActor
		devUserBob   testhelpers.KubeActor
	)

	BeforeEach(func() {
		adminUser = testhelpers.NewKubeDefaultActor()

		// TODO default namespace
		devUserAliceToken := GetToken(Params.UaaLocation, "alice", Params.DeveloperPassword)
		devUserAlice = testhelpers.NewKubeActor("alice", devUserAliceToken)

		devUserBobToken := GetToken(Params.UaaLocation, "bob", Params.DeveloperPassword)
		devUserBob = testhelpers.NewKubeActor("bob", devUserBobToken)

		projectName = fmt.Sprintf("my-project-%d", time.Now().UnixNano())
		otherProjectName = fmt.Sprintf("my-other-project-%d", time.Now().UnixNano())
	})

	When("alice and bob have not been given access to a Project", func() {
		It("does not permit them to interact with allowed resources", func() {
			output, err := devUserBob.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("bob"))))
		})
	})

	When("alice and bob have been given User access to a Project", func() {
		var projectResource string

		BeforeEach(func() {
			projectResource = fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, projectName, userName("bob"), userName("alice"))

			adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			DeleteProject(adminUser, projectName)
		})

		It("permits alice and bob to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := devUserAlice.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-alice")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "create", "serviceaccount", "test-sa-bob")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := devUserAlice.RunKubeCtl("-n", projectName, "get", "configmaps")
				return output
			}).Should(SatisfyAll(
				ContainSubstring("test-map-alice"),
			))
		})

		It("does not permit alice or bob to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := devUserAlice.RunKubeCtl("-n", projectName, "create", "quota", "test-quota-alice")
				return output
			}).Should(ContainSubstring("forbidden"))

			Consistently(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		When("adminUser revokes access to a project from bob", func() {
			BeforeEach(func() {
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

			It("prevents bob from interacting with the allowed resources", func() {
				Eventually(func() string {
					output, _ := devUserBob.RunKubeCtl("-n", projectName, "get", "configmaps")
					return output
				}).Should(ContainSubstring(fmt.Sprintf(`User "%s" cannot list resource "configmaps"`, userName("bob"))))
			})
		})
	})

	When("the ldap-experts group has been given Group access to a Project", func() {
		var projectResource string

		BeforeEach(func() {
			projectResource = fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: Group
                    name: %s`, projectName, groupName("ldap-experts"))

			adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			DeleteProject(adminUser, projectName)
		})

		It("permits members of the Group to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-bob")
				return output
			}).Should(ContainSubstring("created"))

			Eventually(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "get", "configmaps,serviceaccounts")
				return output
			}).Should(SatisfyAll(
				ContainSubstring("test-map-bob"),
			))
		})

		It("does not permit members of the Group to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "create", "quota", "test-quota-quota")
				return output
			}).Should(ContainSubstring("forbidden"))

			Consistently(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		It("does not permit non-members of the Group to interact with arbitrary resources", func() {
			Consistently(func() string {
				output, _ := devUserAlice.RunKubeCtl("-n", projectName, "get", "quota")
				return output
			}).Should(ContainSubstring("forbidden"))
		})

		When("adminUser revokes access to a project from the ldap-experts group", func() {
			BeforeEach(func() {
				projectResource = fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access: []`, projectName)

				adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
			})

			It("prevents members of the group from interacting with the allowed resources", func() {
				Eventually(func() string {
					output, _ := devUserBob.RunKubeCtl("-n", projectName, "get", "configmaps")
					return output
				}).Should(ContainSubstring(fmt.Sprintf(`User "%s" cannot list resource "configmaps"`, userName("bob"))))
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
			adminUser.MustRunKubectl("create", "namespace", saNamespace)

			saName := fmt.Sprintf("service-account-acceptance-testt%d", time.Now().UnixNano())
			saToken := testhelpers.CreateServiceAccount(saName, saNamespace)

			sa = testhelpers.NewKubeActor(saName, saToken)

			projectResource = fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: ServiceAccount
                    name: %s
                    namespace: %s`, projectName, saName, saNamespace)

			adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
		})

		AfterEach(func() {
			adminUser.MustRunKubectl("delete", "namespace", saNamespace)
			DeleteProject(adminUser, projectName)
		})

		It("permits the ServiceAccount to interact with the allowed resources", func() {
			Eventually(func() string {
				output, _ := sa.RunKubeCtl("-n", projectName, "create", "configmap", "test-map-sa")
				return output
			}).Should(ContainSubstring("created"))
		})
	})

	When("Alana creates a project with no users", func() {
		BeforeEach(func() {
			projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access: []`,
				projectName,
			)

			adminUser.MustKubeCtlApply(projectResource)
		})

		AfterEach(func() {
			DeleteProject(adminUser, projectName)
		})

		It("adds Alana as a user", func() {
			Eventually(func() int {
				output, err := adminUser.RunKubeCtl("get", "project", projectName, "-o", "json")
				if err != nil {
					return 0
				}

				var proj projects.Project
				if err := json.Unmarshal([]byte(output), &proj); err != nil {
					fmt.Println(err.Error())
					return 0
				}

				return len(proj.Spec.Access)
			}).Should(Equal(1))
		})
	})

	When("Alana creates and then deletes the project", func() {
		BeforeEach(func() {
			projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, projectName, userName("bob"), userName("alice"))

			adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))
			adminUser.MustRunKubectl("delete", "-f", AsFile(projectResource))
		})

		It("removes the corresponding namespace and leaves Bob and Alice unable to add resources", func() {
			Eventually(func() string {
				output, _ := adminUser.RunKubeCtl("get", "namespace", projectName)
				return output
			}).Should(ContainSubstring(
				fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
			))

			message, err := devUserBob.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(message).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("bob"))))

			message, err = devUserAlice.RunKubeCtl("create", "configmap", "test-map")
			Expect(err).To(HaveOccurred())
			Expect(message).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "configmaps"`, userName("alice"))))
		})
	})

	When("Alana tries to create a project over an existing namespace", func() {
		BeforeEach(func() {
			adminUser.MustRunKubectl("create", "namespace", projectName)

			Eventually(func() error {
				_, err := adminUser.RunKubeCtl("get", "namespace", projectName)
				return err
			}).Should(Succeed())
		})

		AfterEach(func() {
			adminUser.MustRunKubectl("delete", "namespace", projectName)
		})

		It("returns an error immediately", func() {
			projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, projectName, userName("bob"))

			reason, err := adminUser.RunKubeCtl("apply", "-f", AsFile(projectResource))
			Expect(err).To(HaveOccurred())

			Expect(reason).To(ContainSubstring(fmt.Sprintf("cannot create project over existing namespace '%s'", projectName)))
		})
	})

	When("an object inside the project namespace won't delete", func() {
		BeforeEach(func() {
			projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                  name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, projectName, userName("bob"))

			adminUser.MustRunKubectl("apply", "-f", AsFile(projectResource))

			podResource := `
                apiVersion: v1
                kind: Pod
                metadata:
                  name: pod-that-wont-delete
                  finalizers:
                  - pdc/dont-delete
                spec:
                  containers:
                  - image: busybox
                    name: busybox`

			Eventually(func() string {
				output, _ := devUserBob.RunKubeCtl("-n", projectName, "apply", "-f", AsFile(podResource))
				return output
			}).Should(ContainSubstring("created"))
		})

		JustBeforeEach(func() {
			adminUser.MustRunKubectl("delete", "project", projectName, "--wait=false")
		})

		AfterEach(func() {
			podResource := `
                apiVersion: v1
                kind: Pod
                metadata:
                  name: pod-that-wont-delete
                  finalizers: []
                spec:
                  containers:
                  - image: busybox
                    name: busybox`
			adminUser.MustRunKubectl("-n", projectName, "apply", "-f", AsFile(podResource))

			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "projects")
				return output
			}).ShouldNot(ContainSubstring(projectName))
			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "namespaces")
				return output
			}).ShouldNot(ContainSubstring(projectName))
		})

		It("prevents the deletion of the project until the object is cleaned up", func() {
			Consistently(func() string {
				output := adminUser.MustRunKubectl("get", "projects")
				return output
			}).Should(ContainSubstring(projectName))
			Consistently(func() string {
				output := adminUser.MustRunKubectl("get", "namespaces")
				return output
			}).Should(ContainSubstring(projectName))
		})

		It("still allows for creation and deletion of other projects", func() {
			Consistently(func() string {
				output := adminUser.MustRunKubectl("get", "projects")
				return output
			}).Should(ContainSubstring(projectName))

			otherProjectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                  name: %s
                spec:
                  access:
                  - kind: User
                    name: %s`, otherProjectName, userName("bob"))

			adminUser.MustRunKubectl("apply", "-f", AsFile(otherProjectResource))

			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "projects")
				return output
			}).Should(ContainSubstring(otherProjectName))
			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "namespaces")
				return output
			}).Should(ContainSubstring(otherProjectName))

			adminUser.MustRunKubectl("delete", "project", otherProjectName)

			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "projects")
				return output
			}).ShouldNot(ContainSubstring(otherProjectName))
			Eventually(func() string {
				output := adminUser.MustRunKubectl("get", "namespaces")
				return output
			}).ShouldNot(ContainSubstring(otherProjectName))
		})
	})

	It("does not allow unknown access types", func() {
		projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: project
                spec:
                  access:
                  - kind: SomeUnknownKind
                    name: %s`, userName("alice"))

		message, err := adminUser.RunKubeCtl("apply", "-f", AsFile(projectResource))
		Expect(err).To(HaveOccurred(), message)
		Expect(message).To(ContainSubstring("spec.access.kind: Unsupported value: \"SomeUnknownKind\": supported values: \"ServiceAccount\", \"User\", \"Group\""))
	})

	It("does not allow alice or bob to create projects", func() {
		projectResource := fmt.Sprintf(`
                apiVersion: projects.vmware.com/v1alpha1
                kind: Project
                metadata:
                 name: %s
                spec:
                  access:
                  - kind: User
                    name: %s
                  - kind: User
                    name: %s`, "project", userName("bob"), userName("alice"))

		output, err := devUserBob.RunKubeCtl("create", "-f", AsFile(projectResource))
		Expect(err).To(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf(`User "%s" cannot create resource "projects"`, userName("bob"))))
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
