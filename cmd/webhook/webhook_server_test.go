package main_test

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal/projects-operator/testhelpers"
)

var _ = PDescribe("Webhook Server", func() {
	var (
		tmpDir           string
		env              []string
		serverSession    *Session
		keyPath, crtPath string
	)

	BeforeEach(func() {
		var err error

		tmpDir, err = ioutil.TempDir("", "webhook-tests")
		Expect(err).NotTo(HaveOccurred())

		keyPath, crtPath = generateTLSKeyAndCert("test", tmpDir)

		env = []string{
			"TLS_KEY_FILEPATH=" + keyPath,
			"TLS_CERT_FILEPATH=" + crtPath,
		}
	})

	JustBeforeEach(func() {
		serverSession = startWebhookServer(env...)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
		serverSession.Kill()
	})

	It("responds to requests on /projects", func() {
		// wait for the server to start listening
		Eventually(func() error {
			_, err := net.Dial("tcp", ":8080")
			return err
		}).Should(Succeed())

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		client := &http.Client{
			Transport: tr,
		}

		req := testhelpers.NewRequestForWebhookAPI(http.MethodPost, "https://localhost:8080/projects")

		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	When("the TLS cert file does not exist", func() {
		BeforeEach(func() {
			env = []string{
				"TLS_CERT_FILEPATH=nothere",
				"TLS_KEY_FILEPATH=" + keyPath,
			}
		})

		It("logs an error message and exits 1", func() {
			Eventually(serverSession.Err).Should(Say("open nothere: no such file or directory"))
			Eventually(serverSession).Should(Exit(1))
		})
	})

	When("the TLS key file does not exist", func() {
		BeforeEach(func() {
			env = []string{
				"TLS_CERT_FILEPATH=" + crtPath,
				"TLS_KEY_FILEPATH=nothere",
			}
		})

		It("logs an error message and exits 1", func() {
			Eventually(serverSession.Err).Should(Say("open nothere: no such file or directory"))
			Eventually(serverSession).Should(Exit(1))
		})
	})
})
