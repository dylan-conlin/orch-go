package main

import (
	"encoding/json"
	"net/http"
)

func jsonWithStatus(w http.ResponseWriter, status int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	return json.NewEncoder(w).Encode(payload)
}

func jsonOK(w http.ResponseWriter, payload interface{}) error {
	return jsonWithStatus(w, http.StatusOK, payload)
}

func jsonErr(w http.ResponseWriter, status int, message string) {
	_ = jsonWithStatus(w, status, map[string]string{"error": message})
}

func requireMethod(w http.ResponseWriter, r *http.Request, expected string) bool {
	if r.Method != expected {
		jsonErr(w, http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

func methodRouter(w http.ResponseWriter, r *http.Request, handlers map[string]http.HandlerFunc) {
	handler, ok := handlers[r.Method]
	if !ok {
		jsonErr(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	handler(w, r)
}
