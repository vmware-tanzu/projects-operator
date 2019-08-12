package acceptance_test

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
)

func TestAcceptance(t *testing.T) {
	SetDefaultEventuallyTimeout(time.Minute)

	BeforeSuite(func() {
		err := env.Parse(&Params)
		if err != nil {
			panic(err)
		}

	})

	BeforeEach(func() {
		RunMake("install")
		RunMake("clean-crs")
		startController()
	})

	AfterEach(func() {
		stopController()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Acceptance Suite")
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

type Kubectl = func(...string) (string, error)

func GetKubectlFor(api, cluster, token string) Kubectl {
	return func(args ...string) (string, error) {
		allArgs := append([]string{"--insecure-skip-tls-verify", "--cluster=" + cluster, "--token=" + token, "--server=" + api}, args...)
		return runKubectl(allArgs...)
	}
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

	return responseMap["id_token"].(string)
}
