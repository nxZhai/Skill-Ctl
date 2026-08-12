package sources

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/scanner"
)

func TestAddBindsExistingCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	paths := model.Paths{
		ReposDir:  filepath.Join(root, "repos"),
		SkillsDir: filepath.Join(root, "skills"),
	}
	upstream := filepath.Join(root, "upstream")
	origin := filepath.Join(root, "origin.git")
	checkout := filepath.Join(paths.ReposDir, "demo")

	git(t, root, "init", upstream)
	configureGitUser(t, upstream)
	if err := os.MkdirAll(filepath.Join(upstream, "skills", "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(upstream, "skills", "example", "SKILL.md"), []byte("---\nname: Example\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, upstream, "add", ".")
	git(t, upstream, "commit", "-m", "add skill")
	git(t, upstream, "branch", "-M", "main")
	git(t, root, "init", "--bare", origin)
	git(t, upstream, "remote", "add", "origin", origin)
	git(t, upstream, "push", "-u", "origin", "main")
	if err := os.MkdirAll(paths.ReposDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "clone", "--branch", "main", origin, checkout)

	db, err := database.Open(filepath.Join(root, "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	view, skills, err := New(db, paths).Add(context.Background(), "demo", origin, "main")
	if err != nil {
		t.Fatal(err)
	}
	if view.CheckoutPath != checkout {
		t.Fatalf("checkout path = %q, want %q", view.CheckoutPath, checkout)
	}
	if len(skills) != 1 || skills[0].ID != "demo::skills/example" {
		t.Fatalf("skills = %#v", skills)
	}
	if _, err := db.GetSource("demo"); err != nil {
		t.Fatalf("source was not registered: %v", err)
	}
}

func TestViewTreatsSymlinkedCheckoutAsLocalSource(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	repo := filepath.Join(root, "development-repo")
	checkout := filepath.Join(root, "repos", "demo")
	git(t, root, "init", repo)
	configureGitUser(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "README.md")
	git(t, repo, "commit", "-m", "initial")
	git(t, repo, "branch", "-M", "self")
	git(t, repo, "remote", "add", "origin", "git@github.com:acme/demo.git")
	if err := os.MkdirAll(filepath.Dir(checkout), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repo, checkout); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(filepath.Join(root, "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	src := model.Source{ID: "demo", URL: "git@github.com:acme/demo.git", Branch: "main", CheckoutPath: checkout, RemoteSHA: "stale", CreatedAt: database.Now()}
	if err := db.InsertSource(src); err != nil {
		t.Fatal(err)
	}

	view, err := New(db, model.Paths{}).View(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if !view.LocalSource {
		t.Fatal("expected local source")
	}
	wantPath, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	if view.LocalPath != wantPath {
		t.Fatalf("local path = %q, want %q", view.LocalPath, wantPath)
	}
	if view.LocalBranch != "self" {
		t.Fatalf("local branch = %q, want self", view.LocalBranch)
	}
	if view.LocalSHA == "" {
		t.Fatal("expected local commit SHA")
	}
	if view.RemoteSHA != "" || len(view.Remotes) != 0 {
		t.Fatalf("expected no remote metadata, got remote_sha=%q remotes=%#v", view.RemoteSHA, view.Remotes)
	}
	if view.Status != "Local source" {
		t.Fatalf("status = %q, want Local source", view.Status)
	}
}

func TestRemoveDeletesManagedLinksAndRecordsButKeepsCheckout(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{
		ReposDir:  filepath.Join(root, "repos"),
		SkillsDir: filepath.Join(root, "skills"),
	}
	db, err := database.Open(filepath.Join(root, "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	checkout := filepath.Join(paths.ReposDir, "demo")
	skillRoot := filepath.Join(checkout, "skills", "example")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	src := model.Source{
		ID:           "demo",
		URL:          "https://github.com/acme/demo.git",
		Branch:       "main",
		CheckoutPath: checkout,
		CreatedAt:    database.Now(),
	}
	if err := db.InsertSource(src); err != nil {
		t.Fatal(err)
	}
	skill := model.Skill{
		ID:           "demo::skills/example",
		SourceID:     "demo",
		RelativePath: "skills/example",
		Name:         "Example",
		DiscoveredAt: database.Now(),
	}
	if err := db.ReplaceSkillsForSource(src.ID, []model.Skill{skill}); err != nil {
		t.Fatal(err)
	}

	manager := New(db, paths)
	if err := manager.Scanner.RebuildUnifiedLinks(src, []model.Skill{skill}); err != nil {
		t.Fatal(err)
	}
	activationLink := filepath.Join(root, "agent-skills", "example")
	if err := os.MkdirAll(filepath.Dir(activationLink), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(scanner.UnifiedSkillPath(paths, skill), activationLink); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertActivation(model.Activation{
		SkillID:   skill.ID,
		Agent:     "codex",
		Scope:     "user",
		LinkPath:  activationLink,
		CreatedAt: database.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if err := manager.Remove(src.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(checkout); err != nil {
		t.Fatalf("checkout should remain: %v", err)
	}
	if _, err := os.Lstat(activationLink); !os.IsNotExist(err) {
		t.Fatalf("activation link should be removed, got err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(paths.SkillsDir, src.ID)); !os.IsNotExist(err) {
		t.Fatalf("unified source directory should be removed, got err=%v", err)
	}
	if _, err := db.GetSource(src.ID); !database.IsNotFound(err) {
		t.Fatalf("source record should be removed, got err=%v", err)
	}
	if _, err := db.GetSkill(skill.ID); !database.IsNotFound(err) {
		t.Fatalf("skill record should be removed, got err=%v", err)
	}
}

func TestRemoveRefusesNonSymlinkUnifiedEntry(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{SkillsDir: filepath.Join(root, "skills")}
	db, err := database.Open(filepath.Join(root, "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	src := model.Source{
		ID:           "demo",
		URL:          "https://github.com/acme/demo.git",
		Branch:       "main",
		CheckoutPath: filepath.Join(root, "repos", "demo"),
		CreatedAt:    database.Now(),
	}
	if err := db.InsertSource(src); err != nil {
		t.Fatal(err)
	}
	unifiedRoot := filepath.Join(paths.SkillsDir, src.ID)
	if err := os.MkdirAll(unifiedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(unifiedRoot, "keep.txt")
	if err := os.WriteFile(entry, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := New(db, paths).Remove(src.ID); err == nil {
		t.Fatal("expected removal to refuse a non-symlink unified entry")
	}
	if _, err := db.GetSource(src.ID); err != nil {
		t.Fatalf("source record should remain after refusal: %v", err)
	}
	if _, err := os.Stat(entry); err != nil {
		t.Fatalf("non-symlink entry should remain: %v", err)
	}
}

func TestMarkChangedSkillsMatchesSkillDirectoryOnly(t *testing.T) {
	skills := []model.Skill{
		{ID: "demo::skills/foo", RelativePath: "skills/foo"},
		{ID: "demo::skills/foo-bar", RelativePath: "skills/foo-bar"},
	}

	markChangedSkills(skills, []int{0, 1}, []string{"skills/foo/SKILL.md"}, func(skill *model.Skill) {
		skill.LocalChanged = true
	})

	if !skills[0].LocalChanged {
		t.Fatal("expected skills/foo to be marked changed")
	}
	if skills[1].LocalChanged {
		t.Fatal("did not expect skills/foo-bar to be marked changed")
	}
}

func TestParseLocalChangedPathsIncludesRenameSides(t *testing.T) {
	paths := parseLocalChangedPaths(" M skills/foo/SKILL.md\nR  skills/old/SKILL.md -> skills/new/SKILL.md\n?? skills/fresh/\n")
	want := []string{"skills/foo/SKILL.md", "skills/old/SKILL.md", "skills/new/SKILL.md", "skills/fresh"}
	if len(paths) != len(want) {
		t.Fatalf("got %d paths, want %d: %#v", len(paths), len(want), paths)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("path %d = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestViewShowsConfiguredAndUpstreamRemotesWithoutUpdateAvailable(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	upstreamWork := filepath.Join(root, "upstream-work")
	origin := filepath.Join(root, "origin.git")
	checkout := filepath.Join(root, "checkout")
	personal := filepath.Join(root, "personal.git")

	git(t, root, "init", upstreamWork)
	configureGitUser(t, upstreamWork)
	if err := os.WriteFile(filepath.Join(upstreamWork, "README.md"), []byte("upstream\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, upstreamWork, "add", "README.md")
	git(t, upstreamWork, "commit", "-m", "initial")
	git(t, upstreamWork, "branch", "-M", "main")
	git(t, root, "init", "--bare", origin)
	git(t, upstreamWork, "remote", "add", "origin", origin)
	git(t, upstreamWork, "push", "-u", "origin", "main")

	git(t, root, "clone", "--branch", "main", origin, checkout)
	configureGitUser(t, checkout)
	git(t, root, "init", "--bare", personal)
	git(t, checkout, "remote", "add", "personal", personal)
	git(t, checkout, "checkout", "-b", "personal/work")
	if err := os.WriteFile(filepath.Join(checkout, "personal.md"), []byte("local improvement\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, checkout, "add", "personal.md")
	git(t, checkout, "commit", "-m", "personal improvement")
	git(t, checkout, "push", "-u", "personal", "personal/work")

	db, err := database.Open(filepath.Join(root, "skillctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	src := model.Source{
		ID:           "demo",
		URL:          origin,
		Branch:       "main",
		CheckoutPath: checkout,
		CreatedAt:    database.Now(),
	}
	if err := db.InsertSource(src); err != nil {
		t.Fatal(err)
	}

	view, err := New(db, model.Paths{}).View(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "Up to date" {
		t.Fatalf("status = %q, want Up to date", view.Status)
	}
	if view.Message != "" {
		t.Fatalf("message = %q, want empty", view.Message)
	}
	originRemote := findSourceRemote(view.Remotes, "origin", "main")
	if originRemote == nil {
		t.Fatalf("missing origin/main remote: %#v", view.Remotes)
	}
	if originRemote.Ahead != 1 || originRemote.Behind != 0 {
		t.Fatalf("origin/main ahead/behind = %d/%d, want 1/0", originRemote.Ahead, originRemote.Behind)
	}
	personalRemote := findSourceRemote(view.Remotes, "personal", "personal/work")
	if personalRemote == nil {
		t.Fatalf("missing personal/personal/work remote: %#v", view.Remotes)
	}
	if personalRemote.Ahead != 0 || personalRemote.Behind != 0 {
		t.Fatalf("personal/personal-work ahead/behind = %d/%d, want 0/0", personalRemote.Ahead, personalRemote.Behind)
	}
}

func findSourceRemote(remotes []model.SourceRemote, name, branch string) *model.SourceRemote {
	for i := range remotes {
		if remotes[i].Name == name && remotes[i].Branch == branch {
			return &remotes[i]
		}
	}
	return nil
}

func configureGitUser(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "config", "user.email", "skillctl-test@example.com")
	git(t, dir, "config", "user.name", "Skillctl Test")
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
