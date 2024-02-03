package main

import (
	"back_office_cacher/utils"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")
	headers.Add("X-Custom-Header", "custom-value")
	status := http.StatusOK

	utils.MakeHeaders(w, headers, status)

	expectedHeaders := map[string]string{
		"Content-Type":    "application/json",
		"X-Custom-Header": "custom-value",
	}

	for key, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(key)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s: %s, but got: %s", key, expectedValue, actualValue)
		}
	}

	if w.Code != status {
		t.Errorf("Expected status code %d, but got: %d", status, w.Code)
	}
}
