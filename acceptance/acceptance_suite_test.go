package acceptance_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/pivotal/projects-operator/testhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Env struct {
	UaaLocation       string `env:"UAA_LOCATION"`
	DeveloperPassword string `env:"DEVELOPER_PASSWORD"`
	OIDCUserPrefix    string `env:"OIDC_USER_PREFIX"`
	OIDCGroupPrefix   string `env:"OIDC_GROUP_PREFIX"`
}

const (
	testClusterRoleRef      = "acceptance-test-clusterrole"
	testClusterRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: "%s"
rules:
  - apiGroups:
      - "*"
    resources:
      - configmaps
      - serviceaccounts
      - pods
    verbs:
      - "*"
`
)

var (
	Params        Env
	tempFilePaths []string
)

func TestAcceptance(t *testing.T) {
	SetDefaultEventuallyTimeout(time.Minute)

	BeforeSuite(func() {
		err := env.Parse(&Params)
		if err != nil {
			panic(err)
		}

		RunMake("clean-crs")
		testhelpers.NewKubeDefaultActor().MustKubeCtlApply(fmt.Sprintf(testClusterRoleTemplate, testClusterRoleRef))
	})

	AfterEach(func() {
		for _, file := range tempFilePaths {
			err := os.Remove(file)
			Expect(err).NotTo(HaveOccurred())
		}
		tempFilePaths = []string{}
	})

	AfterSuite(func() {
		testhelpers.NewKubeDefaultActor().MustRunKubectl("delete", "clusterrole", testClusterRoleRef)
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
}

func AsFile(content string) string {
	tmpProjectFile, err := ioutil.TempFile("", "project")
	Expect(err).NotTo(HaveOccurred())

	_, err = tmpProjectFile.Write([]byte(content))
	Expect(err).NotTo(HaveOccurred())

	tempFilePaths = append(tempFilePaths, tmpProjectFile.Name())
	return tmpProjectFile.Name()
}

func RunMake(task string) {
	command := exec.Command("make", task)
	command.Dir = filepath.Join("..")
	command.Stdout = GinkgoWriter
	command.Stderr = GinkgoWriter
	Expect(command.Run()).To(Succeed())
}

func GetToken(uaaLocation, user, password string) string {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	resp, err := client.PostForm(uaaLocation+"/oauth/token", url.Values{
		"client_id":     {"pks_cluster_client"},
		"client_secret": {""},
		"grant_type":    {"password"},
		"username":      {user},
		"response_type": {"id_token"},
		"password":      {password},
	})
	Expect(err).NotTo(HaveOccurred())
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	responseMap := make(map[string]interface{})
	err = json.Unmarshal(body, &responseMap)
	Expect(err).NotTo(HaveOccurred())
	Expect(responseMap).To(HaveKey("id_token"))

	return responseMap["id_token"].(string)
}

func DeleteProject(adminUser testhelpers.KubeActor, projectName string) {
	adminUser.MustRunKubectl("delete", "project", projectName)

	Eventually(func() string {
		output, _ := adminUser.RunKubeCtl("get", "namespace", projectName)
		return output
	}).Should(
		ContainSubstring(
			fmt.Sprintf("Error from server (NotFound): namespaces \"%s\" not found", projectName),
		))
}
