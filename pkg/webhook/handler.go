package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
)

// +kubebuilder:rbac:groups=projects.pivotal.io,resources=projectaccesses,verbs=get;create;delete

func NewHandler(logger logr.Logger, namespaceFetcher NamespaceFetcher, projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) http.Handler {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(logger.WithName("project"), namespaceFetcher)
	projectAccessHandler := NewProjectAccessHandler(logger.WithName("projectaccess"), projectFetcher, projectFilterer)

	mux.HandleFunc("/project", projectHandler.HandleProjectValidation)
	mux.HandleFunc("/projectaccess", projectAccessHandler.HandleProjectAccess)
	mux.HandleFunc("/project-create", projectHandler.HandleProjectCreation)

	return mux
}

func ensureBody(bodyReader io.Reader) ([]byte, error) {
	var body []byte

	if bodyReader != nil {
		data, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			return []byte{}, err
		}
		body = data
	}

	return body, nil
}

func unmarshalToAdmissionReview(body []byte) (*admissionv1.AdmissionReview, error) {
	arRequest := &admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, arRequest); err != nil {
		return &admissionv1.AdmissionReview{}, err
	}

	return arRequest, nil
}

func sendReview(w http.ResponseWriter, arReview *admissionv1.AdmissionReview) {
	response, err := json.Marshal(arReview)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error marshalling response": "%s"}`, err))
		return
	}

	_, err = w.Write(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf(`{"error writing response": "%s"}`, err))
		return
	}
}
