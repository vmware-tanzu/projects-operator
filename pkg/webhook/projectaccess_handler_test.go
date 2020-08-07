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
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/testhelpers"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal/projects-operator/pkg/webhook"
	"github.com/pivotal/projects-operator/pkg/webhook/webhookfakes"
)

var _ = Describe("ProjectAccessHandler", func() {
	var (
		responseRecorder *httptest.ResponseRecorder
		h                http.Handler

		fakeProjectFetcher  *webhookfakes.FakeProjectFetcher
		fakeProjectFilterer *webhookfakes.FakeProjectFilterer
	)

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()

		fakeProjectFetcher = new(webhookfakes.FakeProjectFetcher)
		fakeProjectFetcher.GetProjectsReturns([]projects.Project{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-project-a",
				},
				Spec: projects.ProjectSpec{
					Access: []projects.SubjectRef{
						{
							Name: "group-a",
							Kind: "Group",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-project-b",
				},
				Spec: projects.ProjectSpec{
					Access: []projects.SubjectRef{
						{
							Name: "group-b",
							Kind: "Group",
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-project-c",
				},
				Spec: projects.ProjectSpec{
					Access: []projects.SubjectRef{
						{
							Name: "developer",
							Kind: "User",
						},
					},
				},
			},
		}, nil)

		fakeProjectFilterer = new(webhookfakes.FakeProjectFilterer)
		fakeProjectFilterer.FilterProjectsReturns([]string{"my-project-a", "my-project-c"})

		logger := new(testing.NullLogger)
		h = NewHandler(logger, nil, fakeProjectFetcher, fakeProjectFilterer)
	})

	It("handles POST /projectaccess", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess"))

		Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusOK))
	})

	It("fetches all projects", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess"))

		Expect(fakeProjectFetcher.GetProjectsCallCount()).To(Equal(1))
	})

	It("filters the projects for the user", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess"))

		Expect(fakeProjectFilterer.FilterProjectsCallCount()).To(Equal(1))

		projects, user := fakeProjectFilterer.FilterProjectsArgsForCall(0)
		Expect(projects).To(HaveLen(3))
		Expect(user).To(BeEquivalentTo(authenticationv1.UserInfo{Username: "developer", Groups: []string{"group-a"}}))
	})

	It("returns a patched admission review", func() {
		h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess"))

		response, err := ioutil.ReadAll(responseRecorder.Result().Body)
		Expect(err).NotTo(HaveOccurred())

		var admissionReview *admissionv1.AdmissionReview
		Expect(json.Unmarshal(response, &admissionReview)).To(Succeed())
		admissionResponse := admissionReview.Response
		Expect(*admissionResponse.PatchType).To(Equal(admissionv1.PatchTypeJSONPatch))

		var patch []PatchOperation
		Expect(json.Unmarshal(admissionResponse.Patch, &patch)).To(Succeed())
		Expect(patch).To(HaveLen(1))
		patchOperation := patch[0]
		Expect(patchOperation.Path).To(Equal("/status"))
		Expect(patchOperation.Value.(map[string]interface{})["projects"]).To(ConsistOf("my-project-a", "my-project-c"))
	})

	When("the ProjectFetcher returns an error", func() {
		BeforeEach(func() {
			fakeProjectFetcher.GetProjectsReturns([]projects.Project{}, errors.New("error-fetching-projects"))
		})

		It("returns an internal server error", func() {
			h.ServeHTTP(responseRecorder, testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess"))

			body, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(body)).To(ContainSubstring("error fetching projects"))
			Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusInternalServerError))
		})
	})

	When("the request in not json", func() {
		It("returns an invalid request error", func() {
			request := testhelpers.ValidRequestForProjectAccessWebhookAPI(http.MethodPost, "/projectaccess")
			request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte("non-json-body")))

			h.ServeHTTP(responseRecorder, request)

			body, err := ioutil.ReadAll(responseRecorder.Result().Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(body)).To(ContainSubstring("error unmarshalling request body"))
			Expect(responseRecorder.Result().StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
