package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/pivotal/projects-operator/api/v1alpha1"
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

	// 1. Read request body
	var body []byte
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()))
			return
		}
		body = data
	}

	// 2. Unmarshal the body into an AdmissionReview
	arRequest := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &arRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, fmt.Sprintf(`{"error unmarshalling request body": "%s"}`, err))
		return
	}

	// 3. Unmarshal the admissionreview.object.raw into a v1alpha1.ProjectAccess
	raw := arRequest.Request.Object.Raw
	projectAccess := v1alpha1.ProjectAccess{}
	if err := json.Unmarshal(raw, &projectAccess); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, fmt.Sprintf(`{"error unmarshalling ProjectList": "%s"}`, err))
		return
	}

	// 4. Grab the user and groups from the admissionreview.UserInfo
	user := arRequest.Request.UserInfo

	// 5. Grab a list of all projects
	projects, err := h.ProjectFetcher.GetProjects()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error fetching projects": "%s"}`, err.Error()))
		return
	}

	// 6. Do some logic to determine list of projects for the user
	filteredProjects := h.ProjectFilterer.FilterProjects(projects, user)

	// 7. Create a patch to update the status on the incoming ProjectAccess
	patchBytes, err := createPatch(filteredProjects)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error creating ProjectAccess patch": "%s"}`, err.Error()))
		return
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	arResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			Allowed:   true,
			Patch:     patchBytes,
			PatchType: &jsonPatchType,
		},
	}

	// 8. Marshal it
	response, err := json.Marshal(arResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error marshalling response": "%s"}`, err))
		return
	}

	// 9. Write it back
	_, err = w.Write(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error writing response": "%s"}`, err))
		return
	}
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
