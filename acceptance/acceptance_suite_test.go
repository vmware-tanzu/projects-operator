package acceptance_test

import (
	"io"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var controllerSession *Session

func TestAcceptance(t *testing.T) {
	SetDefaultEventuallyTimeout(time.Minute)

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

func RunKubectl(args ...string) string {
	outBuf := NewBuffer()
	command := exec.Command("kubectl", args...)
	command.Stdout = io.MultiWriter(GinkgoWriter, outBuf)
	command.Stderr = io.MultiWriter(GinkgoWriter, outBuf)
	Expect(command.Run()).To(Succeed())

	return string(outBuf.Contents())
}

func RunMake(task string) {
	command := exec.Command("make", task)
	command.Dir = filepath.Join("..")
	command.Stdout = GinkgoWriter
	command.Stderr = GinkgoWriter
	Expect(command.Run()).To(Succeed())
}
