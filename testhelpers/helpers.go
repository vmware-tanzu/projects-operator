// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package testhelpers

// NB: these testhelpers are a copy of the testhelpers from pivotal/developer-console.
// Created https://www.pivotaltracker.com/story/show/169344533 in order to extract and de-dup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	brokerUsername                  = "admin"
	brokerPassword                  = "password"
	DeveloperConsoleSystemNamespace = "pdc-system"
)

// *** general helpers *** //

func PrintTestSetup() {
	fmt.Println("Running tests against the following cluster")
	output := NewKubeDefaultActor().MustRunKubectl("cluster-info")
	fmt.Println(output)
}

func CleanCustomResources() {
	RunMake("clean-crs")
}

// *** test broker helpers *** //

type TestBroker struct {
	ProxySession *Session

	URL      string
	Username string
	Password string
}

func DeployTestBroker(brokerPort int) TestBroker {
	// Ensure there are no leftover/stale overview-broker deployments
	// i.e. from previous test runs
	Eventually(func() string {
		return NewKubeDefaultActor().MustRunKubectl("get", "pod", "-l", "app=overview-broker", "-n", DeveloperConsoleSystemNamespace)
	}).Should(ContainSubstring("No resources found."))

	RunMake("deploy-test-broker")
	NewKubeDefaultActor().MustRunKubectl("wait", "--for=condition=available", "deployment/overview-broker-deployment", "-n", DeveloperConsoleSystemNamespace)
	brokerIP := NewKubeDefaultActor().MustRunKubectl("get", "service", "overview-broker", "-o", "jsonpath={.spec.clusterIP}", "-n", DeveloperConsoleSystemNamespace)

	proxySession := setupProxyAccessToBroker(brokerPort)

	return TestBroker{
		URL:          fmt.Sprintf("http://%s:8080", brokerIP),
		Username:     brokerUsername,
		Password:     brokerPassword,
		ProxySession: proxySession,
	}
}

func DeleteTestBroker(testBroker TestBroker) {
	printBrokerLogs()
	RunMake("delete-test-broker")
	teardownProxyAccessToBroker(testBroker.ProxySession)
}

func printBrokerLogs() {
	fmt.Print("\n\nPrinting broker logs:\n\n")
	fmt.Print(NewKubeDefaultActor().MustRunKubectl("logs", "deployment/overview-broker-deployment", "-n", DeveloperConsoleSystemNamespace))
}

// *** proxy helpers *** //

func setupProxyAccessToBroker(brokerPort int) *Session {
	cmd := exec.Command("kubectl", "-n", DeveloperConsoleSystemNamespace, "port-forward", "service/overview-broker", fmt.Sprintf("%d:8080", brokerPort))

	proxySession, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return proxySession
}

func teardownProxyAccessToBroker(proxySession *Session) {
	proxySession.Terminate()
}

// *** command runners *** //

func RunMake(task string) {
	command := exec.Command("make", task)
	command.Dir = pathToProjectsOperator()
	command.Stdout = GinkgoWriter
	command.Stderr = GinkgoWriter
	Expect(command.Run()).To(Succeed())
}

func RunCLI(pathToCLI string, args []string) string {
	outBuf := NewBuffer()
	command := exec.Command(pathToCLI, args...)
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	command.Run()

	return string(outBuf.Contents())
}

func pathToProjectsOperator() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "..")
}

// TODO: little refactor here
func CreateServiceAccount(serviceAccountName, namespace string) string {
	context := NewKubeDefaultActor()

	message := context.MustRunKubectl("-n", namespace, "create", "serviceaccount", serviceAccountName)
	secretName := context.MustRunKubectl("-n", namespace, "get", "serviceaccount", serviceAccountName, "-o", "jsonpath={.secrets[0].name}")
	secret := context.MustRunKubectl("-n", namespace, "get", "secret", secretName, "-o", "jsonpath={.data.token}")
	token, err := base64.StdEncoding.DecodeString(secret)
	Expect(err).NotTo(HaveOccurred(), message)

	return string(token)
}

type KubeActor struct {
	Name           string
	KubeConfigPath string
	CacheDirPath   string
}

func NewKubeDefaultActor() KubeActor {
	homePath, err := os.UserHomeDir()
	Expect(err).NotTo(HaveOccurred())

	return KubeActor{
		KubeConfigPath: filepath.Join(homePath, ".kube", "config"),
		CacheDirPath:   homePath,
	}
}

func NewKubeActor(name, token string) KubeActor {
	copiedKubeConfigPath := createKubeConfigCopy()

	ka := KubeActor{
		Name:           name,
		KubeConfigPath: copiedKubeConfigPath,
		CacheDirPath:   filepath.Dir(copiedKubeConfigPath),
	}

	_, err := ka.RunKubeCtl("config", "set-credentials", name, "--token="+token)
	Expect(err).NotTo(HaveOccurred())
	_, err = ka.RunKubeCtl("config", "set-context", "--current", "--user="+name)
	Expect(err).NotTo(HaveOccurred())

	return ka
}

func (ka KubeActor) MustRunKubectl(args ...string) string {
	out, err := ka.RunKubeCtl(args...)
	Expect(err).NotTo(HaveOccurred())

	return out
}

func (ka KubeActor) MustKubeCtlApply(yaml string) string {
	command := exec.Command("kubectl", "apply", "-f", "-")
	outBuf := NewBuffer()
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	command.Dir = ka.CacheDirPath
	command.Env = []string{"KUBECONFIG=" + ka.KubeConfigPath}
	command.Stdin = bytes.NewBufferString(yaml)

	Expect(command.Run()).To(Succeed())

	return string(outBuf.Contents())
}

func (ka KubeActor) RunKubeCtl(args ...string) (string, error) {
	command := exec.Command("kubectl", args...)
	outBuf := NewBuffer()
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Dir = ka.CacheDirPath
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	command.Env = []string{"KUBECONFIG=" + ka.KubeConfigPath}
	err := command.Run()
	return string(outBuf.Contents()), err
}

func (ka KubeActor) RunPmCLI(pathToCLI string, args ...string) (string, error) {
	outBuf := NewBuffer()

	command := exec.Command(pathToCLI, args...)
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	command.Env = []string{"KUBECONFIG=" + ka.KubeConfigPath}
	err := command.Run()

	return string(outBuf.Contents()), err
}

func (ka KubeActor) CreateProject(projectName string) {
	ka.MustKubeCtlApply(fmt.Sprintf(`
apiVersion: projects.vmware.com/v1alpha1
kind: Project
metadata:
  name: %s
spec:
  access:
  - kind: ServiceAccount
    name: acceptance-developer
    namespace: test-users-container
`, projectName))
}

func (ka KubeActor) DeleteProject(projectName string) {
	ka.MustRunKubectl("delete", "project", projectName)
}

func (ka KubeActor) UseProject(pathToCLI, projectName string) {
	_, err := ka.RunPmCLI(pathToCLI, "project", "use", "--name", projectName)
	Expect(err).NotTo(HaveOccurred())
}

func createKubeConfigCopy() string {
	copiedKubeConfig, err := ioutil.TempFile("", "kubeactor-config")
	Expect(err).NotTo(HaveOccurred())

	realHomePath, err := os.UserHomeDir()
	Expect(err).NotTo(HaveOccurred())

	contents, err := ioutil.ReadFile(filepath.Join(realHomePath, ".kube", "config"))
	Expect(err).NotTo(HaveOccurred())

	_, err = copiedKubeConfig.Write(contents)
	Expect(err).NotTo(HaveOccurred())

	copiedKubeConfigPath := copiedKubeConfig.Name()
	return copiedKubeConfigPath
}

func ValidRequestForProjectWebhookAPI(method, path, projectName string, requestWithServiceAccount bool) *http.Request {
	project := projects.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
	}
	projectJson, err := json.Marshal(project)
	Expect(err).NotTo(HaveOccurred())

	return requestForWebhookAPI(method, path, projectJson, requestWithServiceAccount)
}

func ValidRequestWithUsersForProjectWebhookAPI(method, path, projectName string) *http.Request {
	project := projects.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
		Spec: projects.ProjectSpec{
			Access: []projects.SubjectRef{
				{
					Kind: rbacv1.UserKind,
					Name: "project-owner",
				},
				{
					Kind: rbacv1.UserKind,
					Name: "project-user",
				},
			},
		},
	}
	projectJson, err := json.Marshal(project)
	Expect(err).NotTo(HaveOccurred())

	return requestForWebhookAPI(method, path, projectJson, false)
}

func ValidRequestForProjectAccessWebhookAPI(method, path string) *http.Request {
	projectAccess := projects.ProjectAccess{}
	projectAccessJson, err := json.Marshal(projectAccess)
	Expect(err).NotTo(HaveOccurred())

	return requestForWebhookAPI(method, path, projectAccessJson, false)
}

func requestForWebhookAPI(method, path string, raw []byte, requestWithServiceAccount bool) *http.Request {
	u, err := url.Parse(path)
	Expect(err).NotTo(HaveOccurred())

	arRequest := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UserInfo: authenticationv1.UserInfo{},
			Object: k8sruntime.RawExtension{
				Raw: raw,
			},
		},
	}

	if requestWithServiceAccount {
		arRequest.Request.UserInfo.Username = "system:serviceaccount:some-namespace:some-serviceaccount"
	} else {
		arRequest.Request.UserInfo.Username = "developer"
		arRequest.Request.UserInfo.Groups = []string{"group-a"}
	}

	body, err := json.Marshal(arRequest)
	Expect(err).NotTo(HaveOccurred())

	return &http.Request{Method: method, URL: u, Body: ioutil.NopCloser(bytes.NewBuffer(body))}
}
