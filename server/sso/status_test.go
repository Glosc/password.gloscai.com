package sso

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSOStatus(t *testing.T) {
	for _, test := range []struct {
		name     string
		register func(*http.ServeMux)
		enabled  bool
	}{
		{"disabled", RegisterDisabled, false},
		{"enabled", func(mux *http.ServeMux) { (&Handler{}).Register(mux) }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			test.register(mux)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/sso/status", nil))
			if response.Code != http.StatusOK || strings.Contains(response.Body.String(), `"enabled":true`) != test.enabled {
				t.Fatalf("unexpected SSO status: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
