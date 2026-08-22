package activation

import (
	"os"
	"path/filepath"
	"testing"

	"skillctl/internal/database"
	"skillctl/internal/model"
)

func TestCleanupDanglingManagedLinks(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "agent", "skills")
	managedDir := filepath.Join(root, "managed")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dangling := filepath.Join(agentDir, "removed")
	if err := os.Symlink(filepath.Join(managedDir, "source", "removed"), dangling); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(agentDir, "foreign")
	if err := os.Symlink(filepath.Join(root, "elsewhere", "missing"), foreign); err != nil {
		t.Fatal(err)
	}
	liveTarget := filepath.Join(managedDir, "source", "live")
	if err := os.MkdirAll(liveTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(agentDir, "live")
	if err := os.Symlink(liveTarget, live); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manager := New(db, model.Paths{SkillsDir: managedDir}, model.Config{
		Agents: map[string]model.AgentConfig{"codex": {UserDir: agentDir}},
	})

	removed, err := manager.CleanupDanglingManagedLinks()
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != dangling {
		t.Fatalf("removed = %#v, want [%q]", removed, dangling)
	}
	if _, err := os.Lstat(dangling); !os.IsNotExist(err) {
		t.Fatalf("dangling link still exists: %v", err)
	}
	if _, err := os.Lstat(foreign); err != nil {
		t.Fatalf("foreign link should remain: %v", err)
	}
	if _, err := os.Lstat(live); err != nil {
		t.Fatalf("live link should remain: %v", err)
	}
}
