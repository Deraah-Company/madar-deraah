package pulse

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPulseRoutesRespectPortalAccess(t *testing.T) {
	s := &Server{
		limit: newLimiter(),
		resolve: func(r *http.Request) (Actor, error) {
			if r.Header.Get("X-Test-Employee") == "yes" {
				return Actor{User: User{ID: "test", Role: "viewer"}}, nil
			}
			return Actor{}, errors.New("no portal session")
		},
	}
	handler := s.routes()
	for _, tc := range []struct {
		method, path string
		employee     bool
		status       int
	}{
		{"GET", "/api/surveys", false, http.StatusUnauthorized},
		{"GET", "/api/surveys", true, http.StatusForbidden},
		{"POST", "/api/surveys", true, http.StatusForbidden},
		{"PATCH", "/api/surveys/example/visibility", true, http.StatusForbidden},
		{"GET", "/api/surveys/example/comments", true, http.StatusForbidden},
		{"POST", "/api/public/cases", false, http.StatusNotFound},
		{"GET", "/api/public/surveys/invalid", false, http.StatusNotFound},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.employee {
			req.Header.Set("X-Test-Employee", "yes")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, w.Code, tc.status)
		}
	}
}
