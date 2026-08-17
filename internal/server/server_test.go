package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadlessRoutesServeAPIOnly(t *testing.T) {
	s := &Server{Token: "test-token"}
	handler := s.apiRoutes()

	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusNotFound {
		t.Fatalf("headless root status = %d, want %d", root.Code, http.StatusNotFound)
	}

	api := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	request.Header.Set("X-Skillctl-Token", "test-token")
	handler.ServeHTTP(api, request)
	if api.Code != http.StatusOK {
		t.Fatalf("headless API status = %d, want %d", api.Code, http.StatusOK)
	}
}

func TestNormalizeSourceNote(t *testing.T) {
	note, err := normalizeSourceNote("  short note\nabout this repo  ")
	if err != nil {
		t.Fatal(err)
	}
	if note != "short note about this repo" {
		t.Fatalf("unexpected normalized note %q", note)
	}
}

func TestNormalizeSourceNoteRejectsMoreThanFiftyWords(t *testing.T) {
	if _, err := normalizeSourceNote(strings.Repeat("word ", sourceNoteWordLimit+1)); err == nil {
		t.Fatal("expected note word limit error")
	}
}

func TestCountSourceNoteWordsCountsCJKCharacters(t *testing.T) {
	if got := countSourceNoteWords("研究工具"); got != 4 {
		t.Fatalf("expected 4 CJK word units, got %d", got)
	}
}
