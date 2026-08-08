package server

import (
	"strings"
	"testing"
)

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
