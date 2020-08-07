// Copyright 2020-Present VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
)

type ProjectAccessHandler struct {
	ProjectFetcher  ProjectFetcher
	ProjectFilterer ProjectFilterer
	logger          logr.Logger
}

func NewProjectAccessHandler(logger logr.Logger, projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) *ProjectAccessHandler {
	return &ProjectAccessHandler{
		ProjectFetcher:  projectFetcher,
		ProjectFilterer: projectFilterer,
		logger:          logger,
	}
}

func (h *ProjectAccessHandler) HandleProjectAccess(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("handling projectaccess request")

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

	// 3. Unmarshal the admissionreview.object.raw into a v1alpha1.ProjectAccess
	raw := arRequest.Request.Object.Raw
	projectAccess := projects.ProjectAccess{}
	if err := json.Unmarshal(raw, &projectAccess); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error unmarshalling ProjectList": "%s"}`, err)

		h.logger.Error(err, "error unmarshaling ProjectAccess from AdmissionReview")
		return
	}

	// 4. Grab the user and groups from the admissionreview.UserInfo
	user := arRequest.Request.UserInfo

	// 5. Grab a list of all projects
	projects, err := h.ProjectFetcher.GetProjects()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error fetching projects": "%s"}`, err.Error())

		h.logger.Error(err, "error fetching Projects")
		return
	}

	// 6. Do some logic to determine list of projects for the user
	filteredProjects := h.ProjectFilterer.FilterProjects(projects, user)

	// 7. Create a patch to update the status on the incoming ProjectAccess
	patchBytes, err := createPatch(filteredProjects)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error creating ProjectAccess patch": "%s"}`, err.Error())

		h.logger.Error(err, "error creating ProjectAccess patch")
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

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func createPatch(projects []string) ([]byte, error) {
	if projects == nil {
		projects = []string{}
	}

	return json.Marshal([]PatchOperation{{
		Op:   "add",
		Path: "/status",
		Value: map[string]interface{}{
			"projects": projects,
		},
	}})
}
