// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-logr/logr"
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectHandler struct {
	NamespaceFetcher NamespaceFetcher
	logger           logr.Logger
}

func NewProjectHandler(logger logr.Logger, namespaceFetcher NamespaceFetcher) *ProjectHandler {
	return &ProjectHandler{
		NamespaceFetcher: namespaceFetcher,
		logger:           logger,
	}
}

func (h *ProjectHandler) HandleProjectValidation(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("handling project request")

	// 1. Read the body
	body, err := ensureBody(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())

		h.logger.Error(err, "error reading body")
		return
	}

	// 2. Unmarshal to AdmissionReview
	arRequest, err := unmarshalToAdmissionReview(body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error unmarshalling request body": "%s"}`, err)

		h.logger.Error(err, "error unmarshaling AdmissionReview")
		return
	}

	// 3. Unmarshal the admissionreview.object.raw into a v1alpha1.Project
	raw := arRequest.Request.Object.Raw
	project := projects.Project{}
	if err := json.Unmarshal(raw, &project); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error unmarshalling project": "%s"}`, err)

		h.logger.Error(err, "error unmarshaling Project from AdmissionReview")
		return
	}

	// 4. Get all current namespaces
	namespaces, err := h.NamespaceFetcher.GetNamespaces()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error fetching namespaces": "%s"}`, err.Error())

		h.logger.Error(err, "error fetching Namespaces")
		return
	}

	// 5. Do some logic to determine if a namespace with the project name already exists
	allowed := true
	for _, namespace := range namespaces {
		if namespace.ObjectMeta.Name == project.ObjectMeta.Name {
			allowed = false
		}
	}

	// 6. Create a response
	arReview := &admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			Allowed: allowed,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: fmt.Sprintf("cannot create project over existing namespace '%s'", project.ObjectMeta.Name),
			},
		},
	}

	// 7. Send AdmissionReview
	sendReview(w, arReview)
}

func (h *ProjectHandler) HandleProjectCreation(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("handling project create request")

	// 1. Read the body
	body, err := ensureBody(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "%s"}`, err.Error())

		h.logger.Error(err, "error reading body")
		return
	}

	arRequest, err := unmarshalToAdmissionReview(body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error unmarshalling request body": "%s"}`, err)

		h.logger.Error(err, "error unmarshaling AdmissionReview")
		return
	}

	raw := arRequest.Request.Object.Raw
	project := projects.Project{}
	if err := json.Unmarshal(raw, &project); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error unmarshalling Project": "%s"}`, err)

		h.logger.Error(err, "error unmarshaling Project from AdmissionReview")
		return
	}

	if len(project.Spec.Access) > 0 {
		sendReview(w, &admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				Allowed: true,
			},
		})
		return
	}

	userInfo := arRequest.Request.UserInfo

	var subjectRef projects.SubjectRef
	if groups := regexp.MustCompile(`system:serviceaccount:([-a-z0-9]+):(.*)`).FindStringSubmatch(userInfo.Username); len(groups) == 3 {
		subjectRef = projects.SubjectRef{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: groups[1],
			Name:      groups[2],
		}
	} else {
		subjectRef = projects.SubjectRef{
			Kind: rbacv1.UserKind,
			Name: userInfo.Username,
		}
	}

	patchBytes, err := createProjectPatch(subjectRef)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error creating Project patch": "%s"}`, err.Error())

		h.logger.Error(err, "error creating Project patch")
		return
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	arReview := &admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			Allowed:   true,
			Patch:     patchBytes,
			PatchType: &jsonPatchType,
		},
	}

	// 8. Send AdmissionReview
	sendReview(w, arReview)
}

func createProjectPatch(user projects.SubjectRef) ([]byte, error) {
	return json.Marshal([]PatchOperation{{
		Op:    "add",
		Path:  "/spec/access",
		Value: interface{}([]projects.SubjectRef{user}),
	}})
}
