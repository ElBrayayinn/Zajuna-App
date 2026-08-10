package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtectLocalAPIRequiresCapabilityForMutation(t *testing.T) {
	handler := protectLocalAPI(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "test-capability")

	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:43123/api/setup", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without capability, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:43123/api/setup", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: capabilityCookieName, Value: "test-capability"})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected request with capability to pass, got %d", response.Code)
	}
}

func TestProtectLocalAPIRejectsCrossOriginAndInvalidContentType(t *testing.T) {
	handler := protectLocalAPI(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "test-capability")

	request := httptest.NewRequest(http.MethodPut, "http://127.0.0.1:43123/api/settings", strings.NewReader(`{}`))
	request.AddCookie(&http.Cookie{Name: capabilityCookieName, Value: "test-capability"})
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected cross-origin request to be rejected first, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPut, "http://127.0.0.1:43123/api/settings", strings.NewReader(`{}`))
	request.AddCookie(&http.Cookie{Name: capabilityCookieName, Value: "test-capability"})
	request.Header.Set("Content-Type", "text/plain")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected invalid content type to be rejected, got %d", response.Code)
	}
}

func TestProtectLocalAPIAllowsEmptyBodyMutation(t *testing.T) {
	handler := protectLocalAPI(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "test-capability")
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:43123/api/backups", nil)
	request.AddCookie(&http.Cookie{Name: capabilityCookieName, Value: "test-capability"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected empty-body action to pass, got %d", response.Code)
	}
}
