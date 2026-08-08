package localskills

import (
	"os"
	"path/filepath"
	"testing"

	"skillctl/internal/model"
)

func TestScanAgentRootFollowsDirectorySymlinks(t *testing.T) {
	tempDir := t.TempDir()
	agentRoot := filepath.Join(tempDir, "agent-skills")
	realSkill := filepath.Join(tempDir, "real-skill")
	if err := os.MkdirAll(agentRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(realSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realSkill, "SKILL.md"), []byte("---\nname: Linked Skill\ndescription: From a linked directory\n---\n# Linked Skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realSkill, filepath.Join(agentRoot, "linked-skill")); err != nil {
		t.Fatal(err)
	}

	items, err := scanAgentRoot("codex", localAgentRoot{Key: "agent-skills", Path: agentRoot})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(items))
	}
	if items[0].ID != "codex::agent-skills::linked-skill" {
		t.Fatalf("unexpected id: %s", items[0].ID)
	}
	if items[0].Name != "Linked Skill" {
		t.Fatalf("unexpected name: %s", items[0].Name)
	}
	if !items[0].IsSymlink {
		t.Fatal("expected linked skill to be marked as a symlink")
	}
	if items[0].SymlinkPath != filepath.Join(agentRoot, "linked-skill") {
		t.Fatalf("unexpected symlink path: %s", items[0].SymlinkPath)
	}
	expectedRealSkill, err := filepath.EvalSymlinks(realSkill)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].RealPath != expectedRealSkill {
		t.Fatalf("unexpected real path: %s", items[0].RealPath)
	}
}

func TestScanAgentRootMarksDirectSkill(t *testing.T) {
	tempDir := t.TempDir()
	skillRoot := filepath.Join(tempDir, "agent-skills", "direct-skill")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("# Direct Skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanAgentRoot("codex", localAgentRoot{Key: "agent-skills", Path: filepath.Join(tempDir, "agent-skills")})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(items))
	}
	if items[0].IsSymlink {
		t.Fatal("expected direct skill not to be marked as a symlink")
	}
	if items[0].SymlinkPath != "" {
		t.Fatalf("unexpected symlink path: %s", items[0].SymlinkPath)
	}
	if items[0].RealPath != "" {
		t.Fatalf("unexpected real path: %s", items[0].RealPath)
	}
}

func TestManagerResolvesLocalSkillID(t *testing.T) {
	tempDir := t.TempDir()
	skillRoot := filepath.Join(tempDir, "skills", "demo")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	manager := New(model.Config{
		Agents: map[string]model.AgentConfig{
			"test-agent": {UserDir: filepath.Join(tempDir, "skills")},
		},
	})
	items, err := manager.List("test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(items))
	}
	content, err := manager.Content(items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Demo\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}
