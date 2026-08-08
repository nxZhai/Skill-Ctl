package appupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"skillctl/internal/model"
)

func TestSnapshotAndBackupIncludeUserData(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{
		ConfigDir: filepath.Join(root, "config"),
		DataDir:   filepath.Join(root, "data"),
		CacheDir:  filepath.Join(root, "cache"),
		ReposDir:  filepath.Join(root, "external-repos"),
		SkillsDir: filepath.Join(root, "external-skills"),
	}
	writeFile(t, filepath.Join(paths.ConfigDir, "config.toml"), "[agents]\n")
	writeFile(t, filepath.Join(paths.DataDir, "state.sqlite"), "database")
	writeFile(t, filepath.Join(paths.ReposDir, "demo", "SKILL.md"), "# Demo\n")
	if err := os.MkdirAll(paths.SkillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(paths.ReposDir, "demo"), filepath.Join(paths.SkillsDir, "demo")); err != nil {
		t.Fatal(err)
	}

	before, err := takeSnapshot(paths)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := createBackup(paths)
	if err != nil {
		t.Fatal(err)
	}
	entries := archiveEntries(t, backup)
	for _, name := range []string{"config/config.toml", "data/state.sqlite", "repos/demo/SKILL.md", "skills/demo"} {
		if !entries[name] {
			t.Fatalf("backup is missing %s", name)
		}
	}
	after, err := takeSnapshot(paths)
	if err != nil {
		t.Fatal(err)
	}
	if !sameSnapshot(before, after) {
		t.Fatal("creating a backup changed user data")
	}
	writeFile(t, filepath.Join(paths.ReposDir, "demo", "SKILL.md"), "# Changed\n")
	changed, err := takeSnapshot(paths)
	if err != nil {
		t.Fatal(err)
	}
	if sameSnapshot(before, changed) {
		t.Fatal("snapshot did not detect a repository change")
	}
}

func TestSelectAssetMatchesCurrentPlatform(t *testing.T) {
	release := Release{TagName: "v0.5.0", Assets: []ReleaseAsset{{
		Name:               "skillctl_0.5.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz",
		BrowserDownloadURL: "https://example.com/skillctl.tar.gz",
	}}}
	asset, err := selectAsset(release)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name != release.Assets[0].Name {
		t.Fatalf("selected %q, want %q", asset.Name, release.Assets[0].Name)
	}
}

func TestVersionComparison(t *testing.T) {
	newer, err := isNewer("v0.5.0", "0.4.1")
	if err != nil || !newer {
		t.Fatalf("expected newer version, got newer=%v err=%v", newer, err)
	}
	newer, err = isNewer("v0.5.0", "0.5.0")
	if err != nil || newer {
		t.Fatalf("expected equal version, got newer=%v err=%v", newer, err)
	}
}

func TestApplyReplacesTemporaryBinaryWithoutChangingUserData(t *testing.T) {
	root := t.TempDir()
	paths := model.Paths{
		ConfigDir: filepath.Join(root, "config"),
		DataDir:   filepath.Join(root, "data"),
		CacheDir:  filepath.Join(root, "cache"),
		ReposDir:  filepath.Join(root, "data", "repos"),
		SkillsDir: filepath.Join(root, "data", "skills"),
	}
	writeFile(t, filepath.Join(paths.ConfigDir, "config.toml"), "[agents]\n")
	writeFile(t, filepath.Join(paths.ReposDir, "demo", "SKILL.md"), "# Demo\n")
	executable := filepath.Join(root, "bin", "skillctl")
	writeFile(t, executable, "old binary")
	archive := releaseArchive(t, "new binary")
	digest := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	name := "skillctl_0.6.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	release := Release{TagName: "v0.6.0", Assets: []ReleaseAsset{{Name: name, BrowserDownloadURL: server.URL, Digest: "sha256:" + hex.EncodeToString(digest[:])}}}
	result, err := Apply(context.Background(), server.Client(), paths, "0.5.0", executable, release)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == "" {
		t.Fatal("update did not create a backup")
	}
	body, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new binary" {
		t.Fatalf("binary was not replaced: %q", body)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func archiveEntries(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	entries := map[string]bool{}
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		entries[header.Name] = true
	}
	return entries
}

func releaseArchive(t *testing.T, body string) []byte {
	t.Helper()
	var output bytes.Buffer
	gz := gzip.NewWriter(&output)
	archive := tar.NewWriter(gz)
	if err := archive.WriteHeader(&tar.Header{Name: "skillctl_0.6.0/skillctl", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
