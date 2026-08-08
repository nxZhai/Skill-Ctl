package appuninstall

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"skillctl/internal/activation"
	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/sources"
)

func TestDisableAllRemovesRecordedManagedLinks(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{SkillsDir: filepath.Join(root, "skills")}
	db, err := database.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	skill := model.Skill{ID: "demo::skill", SourceID: "demo", RelativePath: "skill", Name: "demo", DiscoveredAt: database.Now()}
	if err := db.InsertSource(model.Source{ID: "demo", URL: "https://example.com/demo.git", Branch: "main", CheckoutPath: filepath.Join(root, "repos", "demo"), CreatedAt: database.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceSkillsForSource("demo", []model.Skill{skill}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(paths.SkillsDir, "demo", "skill")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(root, "agent-skills")
	manager := activation.New(db, paths, model.Config{Agents: map[string]model.AgentConfig{"codex": {UserDir: agentDir}}})
	if _, err := manager.Enable(activation.EnableRequest{SkillIDs: []string{skill.ID}, Agent: "codex", Scope: "user"}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(agentDir, "skill")
	if _, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	}
	results, err := DisableAll(db, manager)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("unexpected cleanup results: %#v", results)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("managed link still exists: %v", err)
	}
}

func TestRemoveRepositoriesDeletesOnlyManagedCheckout(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{ReposDir: filepath.Join(root, "repos"), SkillsDir: filepath.Join(root, "skills")}
	db, err := database.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	checkout := filepath.Join(paths.ReposDir, "demo")
	if err := os.MkdirAll(checkout, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "SKILL.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertSource(model.Source{ID: "demo", URL: "https://example.com/demo.git", Branch: "main", CheckoutPath: checkout, CreatedAt: database.Now()}); err != nil {
		t.Fatal(err)
	}
	removed, err := RemoveRepositories(db, sources.New(db, paths), paths.ReposDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != checkout {
		t.Fatalf("unexpected removed paths: %#v", removed)
	}
	if _, err := os.Stat(checkout); !os.IsNotExist(err) {
		t.Fatalf("checkout still exists: %v", err)
	}
}

func TestConfirmAcceptsOnlyYes(t *testing.T) {
	var output bytes.Buffer
	if Confirm(bytes.NewBufferString("no\n"), &output, "Delete?") {
		t.Fatal("no should not confirm")
	}
	if !Confirm(bytes.NewBufferString("yes\n"), &output, "Delete?") {
		t.Fatal("yes should confirm")
	}
}
