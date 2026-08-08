package database

import (
	"path/filepath"
	"testing"

	"skillctl/internal/model"
)

func TestRenameSourceUpdatesSkillsTagsAndActivations(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	oldSource := model.Source{
		ID:           "old-repo",
		URL:          "https://github.com/acme/old-repo.git",
		Branch:       "main",
		CheckoutPath: "/tmp/old-repo",
		CreatedAt:    Now(),
	}
	if err := db.InsertSource(oldSource); err != nil {
		t.Fatal(err)
	}
	oldSkill := model.Skill{
		ID:           "old-repo::skills/demo",
		SourceID:     "old-repo",
		RelativePath: "skills/demo",
		Name:         "Demo",
		DiscoveredAt: Now(),
	}
	if err := db.ReplaceSkillsForSource("old-repo", []model.Skill{oldSkill}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateTags([]string{oldSkill.ID}, []string{"old-repo", "custom"}, "add"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertActivation(model.Activation{
		SkillID:   oldSkill.ID,
		Agent:     "codex",
		Scope:     "user",
		LinkPath:  "/tmp/codex/demo",
		CreatedAt: Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := db.RenameSource("old-repo", "new-repo", "/tmp/new-repo"); err != nil {
		t.Fatal(err)
	}

	source, err := db.GetSource("new-repo")
	if err != nil {
		t.Fatal(err)
	}
	if source.CheckoutPath != "/tmp/new-repo" {
		t.Fatalf("unexpected checkout path: %s", source.CheckoutPath)
	}
	if _, err := db.GetSource("old-repo"); !IsNotFound(err) {
		t.Fatalf("old source should not exist, got err=%v", err)
	}

	skill, err := db.GetSkill("new-repo::skills/demo")
	if err != nil {
		t.Fatal(err)
	}
	if skill.SourceID != "new-repo" {
		t.Fatalf("unexpected source id: %s", skill.SourceID)
	}
	wantTags := map[string]bool{"new-repo": true, "custom": true}
	if len(skill.Tags) != len(wantTags) {
		t.Fatalf("unexpected tags: %#v", skill.Tags)
	}
	for _, tag := range skill.Tags {
		if !wantTags[tag] {
			t.Fatalf("unexpected tag after rename: %s", tag)
		}
	}
	if len(skill.Activations) != 1 {
		t.Fatalf("expected activation to be preserved, got %#v", skill.Activations)
	}
	if skill.Activations[0].SkillID != skill.ID {
		t.Fatalf("activation skill id was not renamed: %#v", skill.Activations[0])
	}
}

func TestPinnedSourcesSortFirst(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, src := range []model.Source{
		{ID: "older", URL: "https://github.com/acme/older.git", Branch: "main", CheckoutPath: "/tmp/older", CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "newer", URL: "https://github.com/acme/newer.git", Branch: "main", CheckoutPath: "/tmp/newer", CreatedAt: "2026-01-02T00:00:00Z"},
	} {
		if err := db.InsertSource(src); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.UpdateSourcePinned("older", true); err != nil {
		t.Fatal(err)
	}

	sources, err := db.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].ID != "older" || !sources[0].Pinned {
		t.Fatalf("expected pinned older source first, got %#v", sources)
	}
	if sources[1].ID != "newer" || sources[1].Pinned {
		t.Fatalf("expected unpinned newer source second, got %#v", sources)
	}
}
