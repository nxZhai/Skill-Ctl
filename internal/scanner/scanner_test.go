package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSkillSearchRootsPrefersConventionalAndPluginSkills(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "skills", "top"))
	mkdir(t, filepath.Join(root, "hf-mcp", "skills", "server"))
	mkdir(t, filepath.Join(root, "plugins", "vllm-skills", "skills", "deploy"))
	mkdir(t, filepath.Join(root, "docs"))

	roots, err := skillSearchRoots(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "skills"),
		filepath.Join(root, "hf-mcp", "skills"),
	}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("unexpected roots:\n got: %#v\nwant: %#v", roots, want)
	}
}

func TestSkillSearchRootsUsesPluginsWhenNoTopLevelSkillRoots(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "plugins", "vllm-skills", "skills", "deploy"))
	mkdir(t, filepath.Join(root, "docs"))

	roots, err := skillSearchRoots(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "plugins", "vllm-skills", "skills"),
	}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("unexpected roots:\n got: %#v\nwant: %#v", roots, want)
	}
}

func TestSkillSearchRootsFallsBackToRepoRoot(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "docs"))

	roots, err := skillSearchRoots(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{root}
	if !reflect.DeepEqual(roots, want) {
		t.Fatalf("unexpected roots:\n got: %#v\nwant: %#v", roots, want)
	}
}

func TestUnifiedEntryRelUsesRootPlaceholder(t *testing.T) {
	if got := UnifiedEntryRel("."); got != "_root" {
		t.Fatalf("unexpected root entry: %q", got)
	}
	if got := UnifiedEntryRel("./"); got != "_root" {
		t.Fatalf("unexpected root entry: %q", got)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
