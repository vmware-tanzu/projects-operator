package acceptance_test

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type Env struct {
	UaaLocation     string `env:"UAA_LOCATION"`
	ClusterLocation string `env:"CLUSTER_LOCATION"`
	CodyPassword    string `env:"CODY_PASSWORD"`
}

var (
	controllerSession *Session
	Params            Env
	tempFilePaths     []string
)

func TestAcceptance(t *testing.T) {

	SetDefaultEventuallyTimeout(time.Minute)

	BeforeSuite(func() {
		err := env.Parse(&Params)
		if err != nil {
			panic(err)
		}

		RunMake("install")
		RunMake("clean-crs")
		startController()
	})

	AfterEach(func() {
		for _, file := range tempFilePaths {
			err := os.Remove(file)
			Expect(err).NotTo(HaveOccurred())
		}
		tempFilePaths = []string{}
	})

	AfterSuite(func() {
		stopController()
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

func startController() {
	pathToController, err := Build("github.com/pivotal-cf/marketplace-project")
	Expect(err).NotTo(HaveOccurred())

	command := exec.Command(pathToController)
	controllerSession, err = Start(command, GinkgoWriter, GinkgoWriter)
	Eventually(controllerSession.Err).Should(Say("starting manager"))

	Expect(err).NotTo(HaveOccurred())
}

func stopController() {
	controllerSession.Terminate()
}

func GetContextFor(api, cluster, token string) KubeContext {
	extraArgs := []string{"--insecure-skip-tls-verify", "--cluster=" + cluster, "--token=" + token, "--server=" + api}
	return KubeContext{extraArgs}
}

func GetContextForAlana() KubeContext {
	return KubeContext{[]string{}}
}

type KubeContext struct {
	extraArgs []string
}

func (c KubeContext) Kubectl(args ...string) (string, error) {
	allArgs := append(c.extraArgs, args...)
	return runKubectl(allArgs...)
}

func (c KubeContext) TryKubectl(args ...string) func() string {
	f := func() string {
		s, _ := c.Kubectl(args...)
		return s
	}
	return f
}

func runKubectl(args ...string) (string, error) {
	outBuf := NewBuffer()
	command := exec.Command("kubectl", args...)
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	err := command.Run()

	return string(outBuf.Contents()), err
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
