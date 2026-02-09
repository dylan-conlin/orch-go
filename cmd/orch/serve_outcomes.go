package main

import (
	"fmt"
	"net/http"
)

func (s *Server) handleOutcomes(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	projectDir := r.URL.Query().Get("project")
	if projectDir == "" {
		var err error
		projectDir, err = s.currentProjectDir()
		if err != nil {
			jsonErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed to resolve project directory: %v", err))
			return
		}
	}

	report, err := buildOutcomeReport(projectDir)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, fmt.Sprintf("Failed to build outcomes report: %v", err))
		return
	}

	if err := jsonOK(w, report); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode outcomes report: %v", err), http.StatusInternalServerError)
		return
	}
}
