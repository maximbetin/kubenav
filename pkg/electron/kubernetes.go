package electron

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kubenav/kubenav/pkg/api"
	"github.com/kubenav/kubenav/pkg/api/middleware"

	"k8s.io/client-go/tools/remotecommand"
)

// requestHandler handles the requests against the Kubernetes API server.
func requestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.Write(w, r, nil)
		return
	}

	var request api.Request
	if r.Body == nil {
		middleware.Errorf(w, r, nil, http.StatusBadRequest, "Request body is empty")
		return
	}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, "Could not decode request body")
		return
	}

	_, clientset, err := client.ConfigClientset(request.Cluster)
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, "Could not create clientset")
		return
	}

	result, err := api.KubernetesRequest(request.Method, request.URL, request.Body, request.Timeout, clientset)
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, fmt.Sprintf("Kubernetes API request failed: %s", err.Error()))
		return
	}

	apiResponse := api.Response{
		Data: strings.TrimSuffix(string(result), "\n"),
	}

	middleware.Write(w, r, apiResponse)
	return
}

// execHandler handles the requests to get a shell into a container.
func execHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		middleware.Write(w, r, nil)
		return
	}

	var request api.Request
	if r.Body == nil {
		middleware.Errorf(w, r, nil, http.StatusBadRequest, "Request body is empty")
		return
	}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, "Could not decode request body")
		return
	}

	config, clientset, err := client.ConfigClientset(request.Cluster)
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, "Could not create clientset")
		return
	}

	sessionID, err := api.GenTerminalSessionID()
	if err != nil {
		middleware.Errorf(w, r, err, http.StatusInternalServerError, "Could not generate terminal session id")
		return
	}

	api.TerminalSessions.Set(sessionID, api.TerminalSession{
		ID:       sessionID,
		Bound:    make(chan error),
		SizeChan: make(chan remotecommand.TerminalSize),
	})

	shell := ""
	go api.WaitForTerminal(config, clientset, &request, shell, sessionID)

	middleware.Write(w, r, api.TerminalResponse{ID: sessionID})
	return
}
