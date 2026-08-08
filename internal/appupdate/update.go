// Package appupdate checks GitHub Releases and safely replaces the running Skillctl binary.
package appupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"skillctl/internal/model"
)

const ReleaseRepository = "nxZhai/Skill-Ctl"

var apiBaseURL = "https://api.github.com"

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

type Result struct {
	ReleaseTag string
	BackupPath string
}

type dataItem struct {
	Name string
	Path string
}

type snapshot map[string]string

// Check returns the latest stable release and whether it is newer than currentVersion.
func Check(ctx context.Context, client *http.Client, currentVersion string) (Release, bool, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseURL+"/repos/"+ReleaseRepository+"/releases/latest", nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "skillctl")
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, false, fmt.Errorf("check releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, false, fmt.Errorf("check releases: GitHub returned %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release); err != nil {
		return Release{}, false, fmt.Errorf("decode latest release: %w", err)
	}
	newer, err := isNewer(release.TagName, currentVersion)
	if err != nil {
		return Release{}, false, err
	}
	return release, newer, nil
}

// Apply backs up user data, installs the supplied release, then verifies data was unchanged.
func Apply(ctx context.Context, client *http.Client, paths model.Paths, currentVersion, executable string, release Release) (Result, error) {
	newer, err := isNewer(release.TagName, currentVersion)
	if err != nil {
		return Result{}, err
	}
	if !newer {
		return Result{}, fmt.Errorf("%s is not newer than %s", release.TagName, currentVersion)
	}
	asset, err := selectAsset(release)
	if err != nil {
		return Result{}, err
	}
	before, err := takeSnapshot(paths)
	if err != nil {
		return Result{}, fmt.Errorf("snapshot user data before update: %w", err)
	}
	backupPath, err := createBackup(paths)
	if err != nil {
		return Result{}, fmt.Errorf("back up user data: %w", err)
	}
	archivePath, err := downloadAsset(ctx, client, asset, paths.CacheDir)
	if err != nil {
		return Result{BackupPath: backupPath}, err
	}
	defer os.Remove(archivePath)
	if err := verifyDigest(archivePath, asset.Digest); err != nil {
		return Result{BackupPath: backupPath}, err
	}
	stagedBinary, err := extractBinary(archivePath, filepath.Dir(executable))
	if err != nil {
		return Result{BackupPath: backupPath}, err
	}
	defer os.Remove(stagedBinary)
	if err := replaceExecutable(stagedBinary, executable); err != nil {
		return Result{BackupPath: backupPath}, err
	}
	after, err := takeSnapshot(paths)
	if err != nil {
		return Result{BackupPath: backupPath}, fmt.Errorf("snapshot user data after update: %w", err)
	}
	if !sameSnapshot(before, after) {
		return Result{BackupPath: backupPath}, errors.New("user data changed during update; the backup was preserved for recovery")
	}
	return Result{ReleaseTag: release.TagName, BackupPath: backupPath}, nil
}

func selectAsset(release Release) (ReleaseAsset, error) {
	version := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	want := fmt.Sprintf("skillctl_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == want && asset.BrowserDownloadURL != "" {
			return asset, nil
		}
	}
	return ReleaseAsset{}, fmt.Errorf("release %s does not include %s", release.TagName, want)
}

func downloadAsset(ctx context.Context, client *http.Client, asset ReleaseAsset, cacheDir string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "updates"), 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "skillctl")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", asset.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: server returned %s", asset.Name, resp.Status)
	}
	temp, err := os.CreateTemp(filepath.Join(cacheDir, "updates"), "release-*.tar.gz")
	if err != nil {
		return "", err
	}
	path := temp.Name()
	if _, err := io.Copy(temp, io.LimitReader(resp.Body, 512<<20)); err != nil {
		temp.Close()
		os.Remove(path)
		return "", err
	}
	if err := temp.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func verifyDigest(path, digest string) error {
	if digest == "" {
		return nil
	}
	if !strings.HasPrefix(digest, "sha256:") {
		return fmt.Errorf("unsupported release asset digest %q", digest)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return err
	}
	if hex.EncodeToString(sum.Sum(nil)) != strings.TrimPrefix(digest, "sha256:") {
		return errors.New("downloaded release asset digest does not match GitHub metadata")
	}
	return nil
}

func extractBinary(archivePath, destinationDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("open release archive: %w", err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read release archive: %w", err)
		}
		if filepath.Base(header.Name) != "skillctl" || header.Typeflag != tar.TypeReg {
			continue
		}
		staged, err := os.CreateTemp(destinationDir, ".skillctl-update-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(staged, io.LimitReader(reader, 512<<20)); err != nil {
			staged.Close()
			os.Remove(staged.Name())
			return "", err
		}
		if err := staged.Chmod(0o755); err != nil {
			staged.Close()
			os.Remove(staged.Name())
			return "", err
		}
		if err := staged.Close(); err != nil {
			os.Remove(staged.Name())
			return "", err
		}
		return staged.Name(), nil
	}
	return "", errors.New("release archive does not contain a regular skillctl binary")
}

func replaceExecutable(stagedBinary, executable string) error {
	info, err := os.Stat(executable)
	if err != nil {
		return fmt.Errorf("inspect current executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("current executable is not a regular file: %s", executable)
	}
	if err := os.Rename(stagedBinary, executable); err != nil {
		return fmt.Errorf("replace current executable: %w", err)
	}
	return nil
}

func createBackup(paths model.Paths) (string, error) {
	dir := filepath.Join(paths.CacheDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, "update-"+time.Now().UTC().Format("20060102T150405")+"-*.tar.gz")
	if err != nil {
		return "", err
	}
	path := file.Name()
	gz := gzip.NewWriter(file)
	archive := tar.NewWriter(gz)
	for _, item := range userDataItems(paths) {
		if err := addToArchive(archive, item); err != nil {
			archive.Close()
			gz.Close()
			file.Close()
			os.Remove(path)
			return "", err
		}
	}
	if err := archive.Close(); err != nil {
		gz.Close()
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err := gz.Close(); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func addToArchive(archive *tar.Writer, item dataItem) error {
	if _, err := os.Lstat(item.Path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(item.Path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(item.Path, path)
		if err != nil {
			return err
		}
		name := item.Name
		if rel != "." {
			name = filepath.ToSlash(filepath.Join(item.Name, rel))
		}
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		header.Name = name
		if info.IsDir() {
			header.Name += "/"
		}
		if err := archive.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(archive, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func takeSnapshot(paths model.Paths) (snapshot, error) {
	result := snapshot{}
	for _, item := range userDataItems(paths) {
		if err := addToSnapshot(result, item); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func addToSnapshot(result snapshot, item dataItem) error {
	if _, err := os.Lstat(item.Path); errors.Is(err, os.ErrNotExist) {
		result[item.Name] = "missing"
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(item.Path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(item.Path, path)
		if err != nil {
			return err
		}
		name := item.Name
		if rel != "." {
			name = filepath.ToSlash(filepath.Join(item.Name, rel))
		}
		switch {
		case info.IsDir():
			result[name] = "directory"
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			result[name] = "symlink:" + target
		case info.Mode().IsRegular():
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			sum := sha256.New()
			_, copyErr := io.Copy(sum, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			result[name] = "file:" + hex.EncodeToString(sum.Sum(nil))
		default:
			return fmt.Errorf("unsupported user data entry: %s", path)
		}
		return nil
	})
}

func userDataItems(paths model.Paths) []dataItem {
	items := []dataItem{{Name: "config", Path: paths.ConfigDir}, {Name: "data", Path: paths.DataDir}}
	if !pathInside(paths.DataDir, paths.ReposDir) {
		items = append(items, dataItem{Name: "repos", Path: paths.ReposDir})
	}
	if !pathInside(paths.DataDir, paths.SkillsDir) {
		items = append(items, dataItem{Name: "skills", Path: paths.SkillsDir})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func pathInside(root, path string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func sameSnapshot(left, right snapshot) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func isNewer(tag, current string) (bool, error) {
	if strings.TrimSpace(current) == "dev" {
		return true, nil
	}
	latest, err := parseVersion(tag)
	if err != nil {
		return false, fmt.Errorf("invalid release tag %q: %w", tag, err)
	}
	installed, err := parseVersion(current)
	if err != nil {
		return false, fmt.Errorf("invalid installed version %q: %w", current, err)
	}
	for i := range latest {
		if latest[i] != installed[i] {
			return latest[i] > installed[i], nil
		}
	}
	return false, nil
}

func parseVersion(raw string) ([3]int, error) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return [3]int{}, errors.New("expected major.minor.patch")
	}
	var parsed [3]int
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return [3]int{}, errors.New("version components must be non-negative integers")
		}
		parsed[i] = number
	}
	return parsed, nil
}
