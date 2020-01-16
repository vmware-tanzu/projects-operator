package main_test

import (
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var pathToWebhook string

func TestWebhook(t *testing.T) {
	BeforeSuite(func() {
		var err error

		pathToWebhook, err = Build("github.com/pivotal/projects-operator/cmd/webhook")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterSuite(func() {
		CleanupBuildArtifacts()
	})

	SetDefaultEventuallyTimeout(time.Minute)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Server Suite")
}

func startWebhookServer(env ...string) *gexec.Session {
	usr, err := user.Current()
	Expect(err).NotTo(HaveOccurred())

	command := exec.Command(pathToWebhook)
	command.Stdout = GinkgoWriter
	command.Stderr = GinkgoWriter
	command.Env = []string{
		"HOME=" + usr.HomeDir,
	}

	if len(env) > 0 {
		command.Env = append(command.Env, env...)
	}

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

func generateTLSKeyAndCert(namePrefix, outputDir string) (string, string) {
	keyPath := filepath.Join(outputDir, fmt.Sprintf("%s.key.pem", namePrefix))
	csrPath := filepath.Join(outputDir, fmt.Sprintf("%s.csr", namePrefix))
	crtPath := filepath.Join(outputDir, fmt.Sprintf("%s.crt.pem", namePrefix))

	genKeyAndCSRCommand := exec.Command(
		"openssl", "req", "-nodes", "-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", csrPath,
		"-subj",
		"/C=GB/ST=London/L=London/O=Global Security/OU=IT Department/CN=localhost",
	)

	genKeyAndCSRCommand.Stdout = GinkgoWriter
	genKeyAndCSRCommand.Stderr = GinkgoWriter
	Expect(genKeyAndCSRCommand.Run()).To(Succeed())

	genCrtCommand := exec.Command(
		"openssl", "x509", "-req", "-days", "365",
		"-in", csrPath,
		"-signkey", keyPath,
		"-out", crtPath,
	)

	genCrtCommand.Stdout = GinkgoWriter
	genCrtCommand.Stderr = GinkgoWriter
	Expect(genCrtCommand.Run()).To(Succeed())

	return keyPath, crtPath
}
