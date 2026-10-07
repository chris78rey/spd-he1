package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCurrentSessionExposesDeleteWorkspacePermission(t *testing.T) {
	for _, canDelete := range []bool{false, true} {
		t.Run(map[bool]string{false: "without_permission", true: "with_permission"}[canDelete], func(t *testing.T) {
			s := &server{sessions: map[string]session{
				"test-session": {username: "personal.oracle", canDeleteWorkspaces: canDelete, expiresAt: time.Now().Add(time.Hour)},
			}}
			request := httptest.NewRequest("GET", "/api/session", nil)
			request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
			response := httptest.NewRecorder()
			s.currentSession(response, request)

			if response.Code != 200 {
				t.Fatalf("currentSession() status = %d, body = %s", response.Code, response.Body.String())
			}
			var result struct {
				Username            string `json:"username"`
				CanDeleteWorkspaces bool   `json:"can_delete_workspaces"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Username != "personal.oracle" || result.CanDeleteWorkspaces != canDelete {
				t.Fatalf("unexpected session response: %+v", result)
			}
		})
	}
}
