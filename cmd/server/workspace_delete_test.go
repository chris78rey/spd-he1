package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeleteObjectionWorkspaceLeavesSourceUntouched(t *testing.T) {
	workspacesDir := filepath.Join(t.TempDir(), "expedientes")
	s := &server{
		workspacesDir: workspacesDir,
		sessions: map[string]session{
			"test-session": {username: "tester", canDeleteWorkspaces: true, expiresAt: time.Now().Add(time.Hour)},
		},
	}

	source := stagedJob{ID: "WORK-AMBULATORIO-202608", Service: "AMBULATORIO", Year: "2026", Month: "08"}
	sourceMarker := filepath.Join(s.jobRoot(source.ID), "fuentes", "preservar.txt")
	if err := os.MkdirAll(filepath.Dir(sourceMarker), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceMarker, []byte("primer ingreso intacto"), 0600); err != nil {
		t.Fatal(err)
	}

	id := "JOB-20261006T121831-f86cec050caf7218"
	objectionJob := stagedJob{
		ID: id, Status: "PROCESSED", IsObjections: true,
		ObjectionSourceID: source.ID,
		ObjectionRows:     []objectionRecord{{Tramite: "6141270", Patient: "PACIENTE DE PRUEBA", Matched: true}},
	}
	if err := os.MkdirAll(s.jobRoot(id), 0700); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(objectionJob)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/expedientes/eliminar/"+id,
		bytes.NewBufferString(`{"confirmacion":"`+id+`"}`))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.deleteWorkspace(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("deleteWorkspace() status = %d, body = %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(s.jobRoot(id)); !os.IsNotExist(err) {
		t.Fatalf("objection workspace still exists: stat error = %v", err)
	}
	if content, err := os.ReadFile(sourceMarker); err != nil || string(content) != "primer ingreso intacto" {
		t.Fatalf("source workspace changed after objection deletion: content = %q, error = %v", content, err)
	}
	if !strings.Contains(response.Body.String(), "Oracle no se modificaron") {
		t.Fatalf("delete response did not confirm Oracle/source isolation: %s", response.Body.String())
	}
}

func TestDeleteWorkspaceForbiddenWithoutPermissionLeavesFilesUntouched(t *testing.T) {
	for _, isObjections := range []bool{false, true} {
		name := "reception"
		if isObjections {
			name = "objections"
		}
		t.Run(name, func(t *testing.T) {
			workspacesDir := filepath.Join(t.TempDir(), "expedientes")
			s := &server{
				workspacesDir: workspacesDir,
				sessions: map[string]session{
					"test-session": {username: "externo", canDeleteWorkspaces: false, expiresAt: time.Now().Add(time.Hour)},
				},
			}
			id := "JOB-20261007T163017-2e7af437885fb830"
			if !isObjections {
				id = "WORK-AMBULATORIO-202608"
			}
			job := stagedJob{ID: id, Status: "PROCESSED", IsObjections: isObjections}
			if err := os.MkdirAll(s.jobRoot(id), 0700); err != nil {
				t.Fatal(err)
			}
			metadata, err := json.Marshal(job)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(s.jobRoot(id), "trabajo", "marcador.pdf")
			if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(marker, []byte("preservar"), 0600); err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodDelete, "/api/v1/expedientes/eliminar/"+id,
				bytes.NewBufferString(`{"confirmacion":"`+id+`"}`))
			request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
			response := httptest.NewRecorder()
			s.deleteWorkspace(response, request)

			if response.Code != http.StatusForbidden {
				t.Fatalf("deleteWorkspace() status = %d, want 403, body = %s", response.Code, response.Body.String())
			}
			if content, err := os.ReadFile(marker); err != nil || string(content) != "preservar" {
				t.Fatalf("workspace files changed after denied deletion: content = %q, error = %v", content, err)
			}
			if _, err := os.Stat(filepath.Join(s.jobRoot(id), "job.json")); err != nil {
				t.Fatalf("workspace metadata changed after denied deletion: %v", err)
			}
		})
	}
}

func TestListWorkspacesSkipsHiddenDeletionRecoveryDirectories(t *testing.T) {
	workspacesDir := filepath.Join(t.TempDir(), "expedientes")
	recoveryName := ".JOB-20261006T121831-f86cec050caf7218.delete-recovery"
	recoveryDir := filepath.Join(workspacesDir, recoveryName)
	if err := os.MkdirAll(recoveryDir, 0700); err != nil {
		t.Fatal(err)
	}
	job := stagedJob{
		ID: "JOB-20261006T121831-f86cec050caf7218", Status: "PROCESSED", IsObjections: true,
		ObjectionRows: []objectionRecord{{Tramite: "6141270", Matched: true}},
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recoveryDir, "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}

	s := &server{
		workspacesDir: workspacesDir,
		sessions: map[string]session{
			"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expedientes", nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.listWorkspaces(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("listWorkspaces() status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Workspaces []map[string]any `json:"workspaces"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("hidden recovery directory was returned as a workspace: %+v", result.Workspaces)
	}
}
