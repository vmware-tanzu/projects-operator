// Copyright 2020-Present VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr/testing"
	"github.com/pivotal/projects-operator/testhelpers"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal/projects-operator/pkg/webhook"
	"github.com/pivotal/projects-operator/pkg/webhook/webhookfakes"
)

var _ = Describe("ProjectHandler", func() {
	var (
		responseRecorder *httptest.ResponseRecorder
		h                http.Handler

		fakeNamespaceFetcher *webhookfakes.FakeNamespaceFetcher
		fakeProjectFilterer  *webhookfakes.FakeProjectFilterer
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()

		fakeNamespaceFetcher = new(webhookfakes.FakeNamespaceFetcher)
		fakeNamespaceFetcher.GetNamespacesReturns([]corev1.Namespace{
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-a",
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-b",
				},
			},
		}, nil)

		fakeProjectFilterer = new(webhookfakes.FakeProjectFilterer)
		fakeProjectFilterer.FilterProjectsReturns([]string{"my-project-a", "my-project-c"})

		logger := new(testing.NullLogger)
		h = NewHandler(logger, fakeNamespaceFetcher, nil, nil)
	})

	It("handles POST /project", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-project", false))

		Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusOK))
	})

	It("handles POST /project-create", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project-create", "my-project", false))

		Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusOK))
	})

	It("fetches all namespaces", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-project", false))

		Expect(fakeNamespaceFetcher.GetNamespacesCallCount()).To(Equal(1))
	})

	When("the project name does not match an existing namespace", func() {
		It("permits the admission", func() {
			h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-project", false))

			response, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			var admissionReview *admissionv1.AdmissionReview
			Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())

			Expect(admissionReview.Response.Allowed).To(BeTrue())
		})
	})

	When("the project name does match an existing namespace", func() {
		It("denies the admission", func() {
			h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-namespace-a", false))

			response, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			var admissionReview *admissionv1.AdmissionReview
			Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())

			Expect(admissionReview.Response.Allowed).To(BeFalse())
			Expect(admissionReview.Response.Result.Message).To(Equal("cannot create project over existing namespace 'my-namespace-a'"))
		})
	})

	When("the NamespaceFetcher returns an error", func() {
		BeforeEach(func() {
			fakeNamespaceFetcher.GetNamespacesReturns([]corev1.Namespace{}, errors.New("error-fetching-namespaces"))
		})

		It("returns an internal server error", func() {
			h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-project", false))

			body, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(body)).To(ContainSubstring("error fetching namespaces"))
			Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusInternalServerError))
		})
	})

	When("the request is not json", func() {
		It("returns an invalid request error", func() {
			request := testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project", "my-project", false)
			request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte("non-json-body")))

			h.ServeHTTP(responseRecorder, request)

			body, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(body)).To(ContainSubstring("error unmarshalling request body"))
			Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	When("the project does not have any access defined on the spec during project creation", func() {
		When("the user info has a user", func() {
			It("adds the requesting user as a user on the project", func() {
				h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project-create", "my-project", false))

				response, err := ioutil.ReadAll(responseRecorder.Result().Body)
				Expect(err).NotTo(HaveOccurred())

				var admissionReview *admissionv1.AdmissionReview
				Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())

				Expect(admissionReview.Response.Allowed).To(BeTrue())
				Expect(admissionReview.Response.Patch).To(Equal([]byte(`[{"op":"add","path":"/spec/access","value":[{"kind":"User","name":"developer"}]}]`)))
			})
		})

		When("the user info has a service account", func() {
			It("adds the requesting service account as a user on the project", func() {
				h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectWebhookAPI(http.MethodPost, "/project-create", "my-project", true))

				response, err := ioutil.ReadAll(responseRecorder.Result().Body)
				Expect(err).NotTo(HaveOccurred())

				var admissionReview *admissionv1.AdmissionReview
				Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())

				Expect(admissionReview.Response.Allowed).To(BeTrue())
				Expect(admissionReview.Response.Patch).To(Equal([]byte(`[{"op":"add","path":"/spec/access","value":[{"kind":"ServiceAccount","name":"some-serviceaccount","namespace":"some-namespace"}]}]`)))
			})
		})
	})

	When("the project has access defined on the spec during project creation", func() {
		It("does not modify the project creation request", func() {
			h.ServeHTTP(responseRecorder, testhelpers.ValidRequestWithUsersForProjectWebhookAPI(http.MethodPost, "/project-create", "my-project"))

			response, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			var admissionReview *admissionv1.AdmissionReview
			Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())

			Expect(admissionReview.Response.Allowed).To(BeTrue())
			Expect(admissionReview.Response.Patch).To(BeNil())
		})
	})
})
