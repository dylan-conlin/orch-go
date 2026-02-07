package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireMethodAllowsExpectedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	ok := requireMethod(w, req, http.MethodGet)
	if !ok {
		t.Fatal("expected requireMethod to allow matching method")
	}
}

func TestRequireMethodRejectsUnexpectedMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	ok := requireMethod(w, req, http.MethodGet)
	if ok {
		t.Fatal("expected requireMethod to reject mismatched method")
	}

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON error body: %v", err)
	}
	if body["error"] != "Method not allowed" {
		t.Fatalf("expected method not allowed error, got %q", body["error"])
	}
}

func TestMethodRouterRoutesByMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()

	called := false
	methodRouter(w, req, map[string]http.HandlerFunc{
		http.MethodDelete: func(http.ResponseWriter, *http.Request) {
			called = true
		},
	})

	if !called {
		t.Fatal("expected router to invoke method handler")
	}
}

func TestMethodRouterReturnsMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/", nil)
	w := httptest.NewRecorder()

	methodRouter(w, req, map[string]http.HandlerFunc{
		http.MethodGet: func(http.ResponseWriter, *http.Request) {},
	})

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON error body: %v", err)
	}
	if body["error"] != "Method not allowed" {
		t.Fatalf("expected method not allowed error, got %q", body["error"])
	}
}
