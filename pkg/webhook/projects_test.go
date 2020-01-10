package webhook_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal/projects-operator/pkg/webhook"
)

var _ = Describe("ProjectsHandler", func() {
	var (
		responseRecorder *httptest.ResponseRecorder
		h                http.Handler
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()

		h = NewHandler()
	})

	It("handles GET /projects", func() {
		h.ServeHTTP(responseRecorder, newRequest("GET", "/projects"))

		Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusOK))
	})
})

func newRequest(method, path string) *http.Request {
	u, err := url.Parse(path)
	Expect(err).NotTo(HaveOccurred())

	return &http.Request{Method: method, URL: u}
}
