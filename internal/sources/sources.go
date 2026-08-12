package sources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/scanner"
)

var sourceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type Manager struct {
	DB      *database.DB
	Paths   model.Paths
	Scanner *scanner.Scanner
}

func New(db *database.DB, paths model.Paths) *Manager {
	return &Manager{DB: db, Paths: paths, Scanner: scanner.New(db, paths)}
}

func (m *Manager) Add(ctx context.Context, id, url, branch string) (model.SourceView, []model.Skill, error) {
	id = strings.TrimSpace(id)
	url = strings.TrimSpace(url)
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}
	if url == "" {
		return model.SourceView{}, nil, errors.New("git url is required")
	}
	if id == "" {
		id = localNameFromGitURL(url)
	}
	if id == "" || !sourceIDPattern.MatchString(id) {
		return model.SourceView{}, nil, errors.New("local name must contain only letters, numbers, dot, underscore, or dash")
	}
	if _, err := m.DB.GetSource(id); err == nil {
		return model.SourceView{}, nil, fmt.Errorf("source %q already exists", id)
	} else if !database.IsNotFound(err) {
		return model.SourceView{}, nil, err
	}
	checkout := filepath.Join(m.Paths.ReposDir, id)
	if _, err := os.Stat(checkout); err == nil {
		return m.bindExistingCheckout(ctx, id, url, branch, checkout)
	} else if !errors.Is(err, os.ErrNotExist) {
		return model.SourceView{}, nil, err
	}
	if err := runGit(ctx, "", "clone", "--branch", branch, "--", url, checkout); err != nil {
		return model.SourceView{}, nil, err
	}
	return m.registerCheckout(ctx, id, url, branch, checkout)
}

func (m *Manager) bindExistingCheckout(ctx context.Context, id, url, branch, checkout string) (model.SourceView, []model.Skill, error) {
	originURL, err := gitOutput(ctx, checkout, "remote", "get-url", "origin")
	if err != nil {
		return model.SourceView{}, nil, fmt.Errorf("existing checkout is not a Git repository with an origin remote: %s", checkout)
	}
	if originURL != url {
		return model.SourceView{}, nil, fmt.Errorf("existing checkout origin does not match git url: %s", checkout)
	}
	if _, err := gitOutput(ctx, checkout, "rev-parse", "--verify", "refs/heads/"+branch); err != nil {
		return model.SourceView{}, nil, fmt.Errorf("existing checkout does not contain local branch %q: %s", branch, checkout)
	}
	return m.registerCheckout(ctx, id, url, branch, checkout)
}

func (m *Manager) registerCheckout(ctx context.Context, id, url, branch, checkout string) (model.SourceView, []model.Skill, error) {
	local, _ := gitOutput(ctx, checkout, "rev-parse", "HEAD")
	remote := ""
	if !isLocalSource(checkout) {
		remote, _ = gitOutput(ctx, checkout, "rev-parse", "origin/"+branch)
	}
	src := model.Source{
		ID:           id,
		URL:          url,
		Branch:       branch,
		CheckoutPath: checkout,
		LocalSHA:     local,
		RemoteSHA:    remote,
		LastFetchAt:  database.Now(),
		CreatedAt:    database.Now(),
	}
	if err := m.DB.InsertSource(src); err != nil {
		return model.SourceView{}, nil, err
	}
	skills, err := m.Scanner.ScanSource(src)
	if err != nil {
		return model.SourceView{}, nil, err
	}
	view, err := m.View(ctx, src)
	return view, skills, err
}

func (m *Manager) Rename(ctx context.Context, oldID, newID string) (model.SourceView, []model.Skill, error) {
	oldID = strings.TrimSpace(oldID)
	newID = strings.TrimSpace(newID)
	if newID == "" || !sourceIDPattern.MatchString(newID) {
		return model.SourceView{}, nil, errors.New("local name must contain only letters, numbers, dot, underscore, or dash")
	}
	src, err := m.DB.GetSource(oldID)
	if err != nil {
		return model.SourceView{}, nil, err
	}
	if oldID == newID {
		skills, err := m.DB.ListSkillsBySource(oldID)
		if err != nil {
			return model.SourceView{}, nil, err
		}
		view, err := m.View(ctx, src)
		return view, skills, err
	}
	if _, err := m.DB.GetSource(newID); err == nil {
		return model.SourceView{}, nil, fmt.Errorf("source %q already exists", newID)
	} else if !database.IsNotFound(err) {
		return model.SourceView{}, nil, err
	}

	oldCheckout := filepath.Join(m.Paths.ReposDir, oldID)
	newCheckout := src.CheckoutPath
	movedCheckout := false
	if filepath.Clean(src.CheckoutPath) == filepath.Clean(oldCheckout) {
		newCheckout = filepath.Join(m.Paths.ReposDir, newID)
		if _, err := os.Stat(newCheckout); err == nil {
			return model.SourceView{}, nil, fmt.Errorf("checkout path already exists: %s", newCheckout)
		} else if !errors.Is(err, os.ErrNotExist) {
			return model.SourceView{}, nil, err
		}
		if err := os.Rename(src.CheckoutPath, newCheckout); err != nil {
			return model.SourceView{}, nil, err
		}
		movedCheckout = true
	}

	if err := m.DB.RenameSource(oldID, newID, newCheckout); err != nil {
		if movedCheckout {
			_ = os.Rename(newCheckout, oldCheckout)
		}
		return model.SourceView{}, nil, err
	}

	src.ID = newID
	src.CheckoutPath = newCheckout
	skills, err := m.DB.ListSkillsBySource(newID)
	if err != nil {
		return model.SourceView{}, nil, err
	}
	_ = os.RemoveAll(filepath.Join(m.Paths.SkillsDir, oldID))
	if err := m.Scanner.RebuildUnifiedLinks(src, skills); err != nil {
		return model.SourceView{}, nil, err
	}
	if err := m.refreshActivationLinks(oldID, skills); err != nil {
		return model.SourceView{}, nil, err
	}
	view, err := m.View(ctx, src)
	return view, skills, err
}

func (m *Manager) Remove(id string) error {
	id = strings.TrimSpace(id)
	src, err := m.DB.GetSource(id)
	if err != nil {
		return err
	}
	skills, err := m.DB.ListSkillsBySource(id)
	if err != nil {
		return err
	}

	var activationLinks []string
	for _, skill := range skills {
		expected := scanner.UnifiedSkillPath(m.Paths, skill)
		for _, activation := range skill.Activations {
			exists, err := validateSymlink(activation.LinkPath, expected)
			if err != nil {
				return err
			}
			if exists {
				activationLinks = append(activationLinks, activation.LinkPath)
			}
		}
	}

	unifiedRoot, unifiedLinks, err := m.unifiedSourceLinks(src, skills)
	if err != nil {
		return err
	}
	for _, path := range activationLinks {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	for _, path := range unifiedLinks {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	if err := removeEmptyTree(unifiedRoot); err != nil {
		return err
	}
	return m.DB.DeleteSource(id)
}

func (m *Manager) unifiedSourceLinks(src model.Source, skills []model.Skill) (string, []string, error) {
	root := filepath.Join(m.Paths.SkillsDir, src.ID)
	absSkillsDir, err := filepath.Abs(m.Paths.SkillsDir)
	if err != nil {
		return "", nil, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	if absRoot == absSkillsDir || !strings.HasPrefix(absRoot, absSkillsDir+string(os.PathSeparator)) {
		return "", nil, fmt.Errorf("refusing to remove unsafe unified skill path: %s", root)
	}

	expectedTargets := make(map[string]string, len(skills))
	for _, skill := range skills {
		path := scanner.UnifiedSkillPath(m.Paths, skill)
		expectedTargets[path] = filepath.Join(src.CheckoutPath, filepath.FromSlash(skill.RelativePath))
	}
	if _, err := os.Lstat(absRoot); errors.Is(err, os.ErrNotExist) {
		return absRoot, nil, nil
	} else if err != nil {
		return "", nil, err
	}

	var links []string
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to delete non-symlink unified skill entry: %s", path)
		}
		current, err := os.Readlink(path)
		if err != nil {
			return err
		}
		expected, ok := expectedTargets[path]
		if !ok {
			return fmt.Errorf("refusing to delete unregistered unified skill link: %s", path)
		}
		if current != expected {
			return fmt.Errorf("refusing to delete unified skill link with unexpected target: %s", path)
		}
		links = append(links, path)
		return nil
	})
	return absRoot, links, err
}

func validateSymlink(path, expected string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, fmt.Errorf("refusing to delete non-symlink: %s", path)
	}
	current, err := os.Readlink(path)
	if err != nil {
		return false, err
	}
	if current != expected {
		return false, fmt.Errorf("refusing to delete symlink with unexpected target: %s", path)
	}
	return true, nil
}

func removeEmptyTree(root string) error {
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Remove(dirs[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func localNameFromGitURL(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimRight(value, "/")
	if value == "" {
		return ""
	}
	var name string
	if strings.Contains(value, ":") && !strings.Contains(value, "://") {
		parts := strings.Split(value, ":")
		name = parts[len(parts)-1]
	} else {
		name = value
	}
	parts := strings.Split(name, "/")
	name = parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, "-")
}

func (m *Manager) SkillContent(id string) (string, error) {
	root, err := m.skillRoot(id)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (m *Manager) SkillTree(id string) (model.SkillTree, error) {
	root, err := m.skillRoot(id)
	if err != nil {
		return model.SkillTree{}, err
	}
	entries, err := readTree(root, root, 0)
	if err != nil {
		return model.SkillTree{}, err
	}
	return model.SkillTree{Root: root, Entries: entries}, nil
}

func (m *Manager) OpenSkillDir(id, relPath string) error {
	root, err := m.skillRoot(id)
	if err != nil {
		return err
	}
	target, err := cleanSkillSubdir(root, relPath)
	if err != nil {
		return err
	}
	return exec.Command("open", target).Start()
}

func (m *Manager) OpenSkillPath(id, relPath string) error {
	root, err := m.skillRoot(id)
	if err != nil {
		return err
	}
	target, err := cleanSkillPath(root, relPath)
	if err != nil {
		return err
	}
	return openFileInPreferredEditor(target)
}

func (m *Manager) OpenSourceDir(id string) error {
	src, err := m.DB.GetSource(id)
	if err != nil {
		return err
	}
	info, err := os.Stat(src.CheckoutPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source checkout path is not a directory: %s", src.CheckoutPath)
	}
	return exec.Command("open", src.CheckoutPath).Start()
}

func (m *Manager) skillRoot(id string) (string, error) {
	skill, err := m.DB.GetSkill(id)
	if err != nil {
		return "", err
	}
	src, err := m.DB.GetSource(skill.SourceID)
	if err != nil {
		return "", err
	}
	root := filepath.Join(src.CheckoutPath, filepath.FromSlash(skill.RelativePath))
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("skill path is not a directory: %s", root)
	}
	return root, nil
}

func readTree(root, dir string, depth int) ([]model.SkillTreeEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]model.SkillTreeEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if shouldSkipTreeEntry(name, de.IsDir()) {
			continue
		}
		fullPath := filepath.Join(dir, name)
		rel, err := filepath.Rel(root, fullPath)
		if err != nil {
			return nil, err
		}
		entry := model.SkillTreeEntry{
			Name: name,
			Path: filepath.ToSlash(rel),
			Kind: "file",
		}
		if de.IsDir() {
			entry.Kind = "dir"
			if depth < 3 {
				children, err := readTree(root, fullPath, depth+1)
				if err != nil {
					return nil, err
				}
				entry.Children = children
			}
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind == "dir"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

func shouldSkipTreeEntry(name string, isDir bool) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	if isDir && skipTreeDirs[name] {
		return true
	}
	return false
}

var skipTreeDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

func cleanSkillSubdir(root, relPath string) (string, error) {
	absTarget, err := cleanSkillPath(root, relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absTarget)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return absTarget, nil
}

func cleanSkillPath(root, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	relPath = strings.TrimPrefix(filepath.ToSlash(relPath), "/")
	cleaned := filepath.Clean(filepath.FromSlash(relPath))
	if cleaned == "." {
		cleaned = ""
	}
	target := filepath.Join(root, cleaned)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return "", errors.New("path is outside the skill")
	}
	if _, err := os.Stat(absTarget); err != nil {
		return "", err
	}
	return absTarget, nil
}

func openFileInPreferredEditor(path string) error {
	switch runtime.GOOS {
	case "darwin":
		for _, app := range []string{"Cursor", "Visual Studio Code", "TextEdit"} {
			if err := exec.Command("open", "-a", app, path).Run(); err == nil {
				return nil
			}
		}
		return exec.Command("open", path).Start()
	case "windows":
		for _, cmd := range []string{"cursor", "code", "notepad"} {
			if _, err := exec.LookPath(cmd); err == nil {
				return exec.Command(cmd, path).Start()
			}
		}
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		for _, cmd := range []string{"cursor", "code"} {
			if _, err := exec.LookPath(cmd); err == nil {
				return exec.Command(cmd, path).Start()
			}
		}
		return exec.Command("xdg-open", path).Start()
	}
}

func (m *Manager) refreshActivationLinks(oldSourceID string, skills []model.Skill) error {
	for _, skill := range skills {
		target := scanner.UnifiedSkillPath(m.Paths, skill)
		oldTarget := filepath.Join(m.Paths.SkillsDir, oldSourceID, filepath.FromSlash(scanner.UnifiedEntryRel(skill.RelativePath)))
		for _, activation := range skill.Activations {
			info, err := os.Lstat(activation.LinkPath)
			if errors.Is(err, os.ErrNotExist) {
				if err := os.MkdirAll(filepath.Dir(activation.LinkPath), 0o755); err != nil {
					return err
				}
				if err := os.Symlink(target, activation.LinkPath); err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return fmt.Errorf("refusing to replace non-symlink activation link: %s", activation.LinkPath)
			}
			current, err := os.Readlink(activation.LinkPath)
			if err != nil {
				return err
			}
			if current == target {
				continue
			}
			if current != oldTarget {
				return fmt.Errorf("refusing to replace symlink with unexpected target: %s", activation.LinkPath)
			}
			if err := os.Remove(activation.LinkPath); err != nil {
				return err
			}
			if err := os.Symlink(target, activation.LinkPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) List(ctx context.Context) ([]model.SourceView, error) {
	sources, err := m.DB.ListSources()
	if err != nil {
		return nil, err
	}
	out := make([]model.SourceView, 0, len(sources))
	for _, src := range sources {
		view, err := m.View(ctx, src)
		if err != nil {
			view = model.SourceView{Source: src, Status: "Not checked", Message: err.Error()}
		}
		out = append(out, view)
	}
	return out, nil
}

func (m *Manager) View(ctx context.Context, src model.Source) (model.SourceView, error) {
	count, err := m.DB.CountSkillsBySource(src.ID)
	if err != nil {
		return model.SourceView{}, err
	}
	view := model.SourceView{Source: src, SkillCount: count, Status: "Not checked"}
	if isLocalSource(src.CheckoutPath) {
		view.LocalSource = true
		view.RemoteSHA = ""
		view.LocalPath, _ = filepath.EvalSymlinks(src.CheckoutPath)
		view.LocalBranch, _ = gitOutput(ctx, src.CheckoutPath, "branch", "--show-current")
		view.LocalSHA, _ = gitOutput(ctx, src.CheckoutPath, "rev-parse", "HEAD")
		view.LastCommitAt, _ = gitOutput(ctx, src.CheckoutPath, "show", "-s", "--format=%cI", "HEAD")
		view.Status = "Local source"
		return view, nil
	}
	if local, err := gitOutput(ctx, src.CheckoutPath, "rev-parse", "HEAD"); err == nil {
		view.LocalSHA = local
	}
	if remotes, err := sourceRemotes(ctx, src); err == nil {
		view.Remotes = remotes
		for _, remote := range remotes {
			if remote.Branch == src.Branch && remote.SHA != "" {
				view.RemoteSHA = remote.SHA
				view.Ahead = remote.Ahead
				view.Behind = remote.Behind
				break
			}
		}
	}
	if lastCommitAt, err := sourceLastCommitAt(ctx, src.CheckoutPath, src.Branch); err == nil {
		view.LastCommitAt = lastCommitAt
	}
	if len(view.Remotes) > 0 {
		view.Status = "Up to date"
		for _, remote := range view.Remotes {
			if remote.Behind > 0 {
				view.Status = "Update available"
				break
			}
		}
	} else if view.LocalSHA != "" && view.RemoteSHA != "" {
		if view.LocalSHA == view.RemoteSHA {
			view.Status = "Up to date"
		} else {
			view.Status = "Update available"
		}
	}
	if ok, err := hasLocalChanges(ctx, src.CheckoutPath); err == nil && ok {
		view.Status = "Local changes"
	}
	if len(view.Remotes) == 0 {
		ahead, behind, err := aheadBehind(ctx, src.CheckoutPath, "origin", src.Branch)
		if err == nil {
			view.Ahead = ahead
			view.Behind = behind
			if behind > 0 {
				view.Status = "Update available"
			}
		}
	}
	return view, nil
}

func (m *Manager) AnnotateSkillChanges(ctx context.Context, skills []model.Skill) []model.Skill {
	if len(skills) == 0 {
		return skills
	}
	sources, err := m.DB.ListSources()
	if err != nil {
		return skills
	}
	sourceByID := make(map[string]model.Source, len(sources))
	for _, src := range sources {
		sourceByID[src.ID] = src
	}
	indexesBySource := make(map[string][]int)
	for i := range skills {
		indexesBySource[skills[i].SourceID] = append(indexesBySource[skills[i].SourceID], i)
	}
	for sourceID, indexes := range indexesBySource {
		src, ok := sourceByID[sourceID]
		if !ok {
			continue
		}
		if localPaths, err := localChangedPaths(ctx, src.CheckoutPath); err == nil {
			markChangedSkills(skills, indexes, localPaths, func(skill *model.Skill) {
				skill.LocalChanged = true
			})
		}
		if !isLocalSource(src.CheckoutPath) {
			remotePaths, err := remoteChangedPaths(ctx, src.CheckoutPath, src.Branch)
			if err != nil {
				continue
			}
			markChangedSkills(skills, indexes, remotePaths, func(skill *model.Skill) {
				skill.RemoteChanged = true
			})
		}
	}
	return skills
}

func (m *Manager) Check(ctx context.Context, id string) (model.SourceView, error) {
	src, err := m.DB.GetSource(id)
	if err != nil {
		return model.SourceView{}, err
	}
	if isLocalSource(src.CheckoutPath) {
		return m.View(ctx, src)
	}
	if err := runGit(ctx, src.CheckoutPath, "fetch", "--all", "--prune"); err != nil {
		view, _ := m.View(ctx, src)
		view.Status = "Sync failed"
		view.Message = err.Error()
		return view, err
	}
	local, err := gitOutput(ctx, src.CheckoutPath, "rev-parse", "HEAD")
	if err != nil {
		return model.SourceView{}, err
	}
	remote, err := gitOutput(ctx, src.CheckoutPath, "rev-parse", "origin/"+src.Branch)
	if err != nil {
		return model.SourceView{}, err
	}
	if err := m.DB.UpdateSourceSHAs(src.ID, local, remote, database.Now()); err != nil {
		return model.SourceView{}, err
	}
	src.LocalSHA = local
	src.RemoteSHA = remote
	src.LastFetchAt = database.Now()
	return m.View(ctx, src)
}

func (m *Manager) Sync(ctx context.Context, id string) (model.SourceView, []model.Skill, error) {
	src, err := m.DB.GetSource(id)
	if err != nil {
		return model.SourceView{}, nil, err
	}
	if isLocalSource(src.CheckoutPath) {
		skills, err := m.Scanner.ScanSource(src)
		if err != nil {
			return model.SourceView{}, nil, err
		}
		view, err := m.View(ctx, src)
		return view, skills, err
	}
	if changed, err := hasLocalChanges(ctx, src.CheckoutPath); err != nil {
		return model.SourceView{}, nil, err
	} else if changed {
		return model.SourceView{Source: src, Status: "Local changes", Message: "Source contains local changes. Please inspect or restore the repository before syncing."}, nil, errors.New("source contains local changes")
	}
	if err := runGit(ctx, src.CheckoutPath, "fetch", "--prune", "origin"); err != nil {
		return model.SourceView{}, nil, err
	}
	if err := runGit(ctx, src.CheckoutPath, "merge", "--ff-only", "origin/"+src.Branch); err != nil {
		return model.SourceView{}, nil, err
	}
	local, err := gitOutput(ctx, src.CheckoutPath, "rev-parse", "HEAD")
	if err != nil {
		return model.SourceView{}, nil, err
	}
	remote, _ := gitOutput(ctx, src.CheckoutPath, "rev-parse", "origin/"+src.Branch)
	if err := m.DB.UpdateSourceSHAs(src.ID, local, remote, database.Now()); err != nil {
		return model.SourceView{}, nil, err
	}
	src.LocalSHA = local
	src.RemoteSHA = remote
	src.LastFetchAt = database.Now()
	skills, err := m.Scanner.ScanSource(src)
	if err != nil {
		return model.SourceView{}, nil, err
	}
	view, err := m.View(ctx, src)
	return view, skills, err
}

func (m *Manager) CheckAll(ctx context.Context) []model.OperationResult {
	sources, err := m.DB.ListSources()
	if err != nil {
		return []model.OperationResult{{OK: false, Status: "error", Message: err.Error()}}
	}
	var results []model.OperationResult
	for _, src := range sources {
		view, err := m.Check(ctx, src.ID)
		results = append(results, model.OperationResult{
			ID: src.ID, OK: err == nil, Status: view.Status, Message: firstNonEmpty(view.Message, errString(err)),
		})
	}
	return results
}

func (m *Manager) SyncAll(ctx context.Context) []model.OperationResult {
	sources, err := m.DB.ListSources()
	if err != nil {
		return []model.OperationResult{{OK: false, Status: "error", Message: err.Error()}}
	}
	var results []model.OperationResult
	for _, src := range sources {
		view, _, err := m.Sync(ctx, src.ID)
		results = append(results, model.OperationResult{
			ID: src.ID, OK: err == nil, Status: view.Status, Message: firstNonEmpty(view.Message, errString(err)),
		})
	}
	return results
}

func hasLocalChanges(ctx context.Context, dir string) (bool, error) {
	out, err := gitOutput(ctx, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func localChangedPaths(ctx context.Context, dir string) ([]string, error) {
	out, err := gitRawOutput(ctx, dir, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parseLocalChangedPaths(out), nil
}

func remoteChangedPaths(ctx context.Context, dir, branch string) ([]string, error) {
	ref := "origin/" + branch
	if _, err := gitOutput(ctx, dir, "rev-parse", "--verify", ref); err != nil {
		return nil, err
	}
	out, err := gitOutput(ctx, dir, "diff", "--name-only", "HEAD..."+ref)
	if err != nil {
		return nil, err
	}
	return parseChangedPathLines(out), nil
}

func isLocalSource(checkout string) bool {
	info, err := os.Lstat(checkout)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func markChangedSkills(skills []model.Skill, indexes []int, changedPaths []string, mark func(*model.Skill)) {
	if len(changedPaths) == 0 {
		return
	}
	for _, index := range indexes {
		skillPath := normalizeRepoPath(skills[index].RelativePath)
		for _, changedPath := range changedPaths {
			if changedPathMatchesSkill(changedPath, skillPath) {
				mark(&skills[index])
				break
			}
		}
	}
}

func changedPathMatchesSkill(changedPath, skillPath string) bool {
	if strings.TrimSpace(changedPath) == "" {
		return false
	}
	changedPath = normalizeRepoPath(changedPath)
	skillPath = normalizeRepoPath(skillPath)
	if skillPath == "." {
		return true
	}
	return changedPath == skillPath || strings.HasPrefix(changedPath, skillPath+"/")
}

func parseLocalChangedPaths(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if len(line) < 4 {
			continue
		}
		pathText := strings.TrimSpace(line[3:])
		if pathText == "" {
			continue
		}
		for _, path := range strings.Split(pathText, " -> ") {
			if normalized := normalizeGitPath(path); normalized != "" {
				paths = append(paths, normalized)
			}
		}
	}
	return paths
}

func parseChangedPathLines(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if normalized := normalizeGitPath(line); normalized != "" {
			paths = append(paths, normalized)
		}
	}
	return paths
}

func normalizeGitPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, `"`) {
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}
	}
	return normalizeRepoPath(path)
}

func normalizeRepoPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return "."
	}
	return path
}

type gitRemote struct {
	Name string
	URL  string
}

type remoteTarget struct {
	Remote string
	Branch string
}

func sourceRemotes(ctx context.Context, src model.Source) ([]model.SourceRemote, error) {
	remotes, err := gitRemotes(ctx, src.CheckoutPath)
	if err != nil {
		return nil, err
	}
	if len(remotes) == 0 {
		return nil, nil
	}
	configuredRemote := remoteNameForURL(remotes, src.URL)
	if configuredRemote == "" {
		if _, ok := remotes["origin"]; ok {
			configuredRemote = "origin"
		}
	}
	upstreamRemote, upstreamBranch := currentUpstream(ctx, src.CheckoutPath, remotes)

	var targets []remoteTarget
	addTarget := func(remote, branch string) {
		remote = strings.TrimSpace(remote)
		branch = strings.TrimSpace(branch)
		if remote == "" || branch == "" {
			return
		}
		for _, target := range targets {
			if target.Remote == remote && target.Branch == branch {
				return
			}
		}
		targets = append(targets, remoteTarget{Remote: remote, Branch: branch})
	}
	addTarget(configuredRemote, src.Branch)
	addTarget(upstreamRemote, upstreamBranch)

	names := sortedRemoteNames(remotes)
	for _, name := range names {
		if hasTargetForRemote(targets, name) {
			continue
		}
		if src.Branch != "" {
			if _, err := remoteBranchSHA(ctx, src.CheckoutPath, name, src.Branch); err == nil {
				addTarget(name, src.Branch)
				continue
			}
		}
		if defaultBranch := remoteDefaultBranch(ctx, src.CheckoutPath, name, remotes); defaultBranch != "" {
			addTarget(name, defaultBranch)
		}
	}

	out := make([]model.SourceRemote, 0, len(targets))
	for _, target := range targets {
		info, ok := remotes[target.Remote]
		if !ok {
			continue
		}
		item := model.SourceRemote{Name: info.Name, URL: info.URL, Branch: target.Branch}
		if sha, err := remoteBranchSHA(ctx, src.CheckoutPath, target.Remote, target.Branch); err == nil {
			item.SHA = sha
		}
		if ahead, behind, err := aheadBehind(ctx, src.CheckoutPath, target.Remote, target.Branch); err == nil {
			item.Ahead = ahead
			item.Behind = behind
		}
		out = append(out, item)
	}
	return out, nil
}

func gitRemotes(ctx context.Context, dir string) (map[string]gitRemote, error) {
	out, err := gitOutput(ctx, dir, "remote", "-v")
	if err != nil {
		return nil, err
	}
	remotes := map[string]gitRemote{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[2] != "(fetch)" {
			continue
		}
		name := strings.TrimSpace(fields[0])
		url := strings.TrimSpace(fields[1])
		if name == "" || url == "" {
			continue
		}
		remotes[name] = gitRemote{Name: name, URL: url}
	}
	return remotes, nil
}

func remoteNameForURL(remotes map[string]gitRemote, url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	for _, name := range sortedRemoteNames(remotes) {
		if remotes[name].URL == url {
			return name
		}
	}
	return ""
}

func currentUpstream(ctx context.Context, dir string, remotes map[string]gitRemote) (string, string) {
	ref, err := gitOutput(ctx, dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return "", ""
	}
	return splitRemoteBranch(ref, remotes)
}

func splitRemoteBranch(ref string, remotes map[string]gitRemote) (string, string) {
	ref = strings.TrimSpace(ref)
	names := sortedRemoteNames(remotes)
	sort.SliceStable(names, func(i, j int) bool {
		return len(names[i]) > len(names[j])
	})
	for _, name := range names {
		prefix := name + "/"
		if strings.HasPrefix(ref, prefix) {
			return name, strings.TrimPrefix(ref, prefix)
		}
	}
	return "", ""
}

func hasTargetForRemote(targets []remoteTarget, remote string) bool {
	for _, target := range targets {
		if target.Remote == remote {
			return true
		}
	}
	return false
}

func sortedRemoteNames(remotes map[string]gitRemote) []string {
	names := make([]string, 0, len(remotes))
	for name := range remotes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func remoteDefaultBranch(ctx context.Context, dir, remote string, remotes map[string]gitRemote) string {
	ref, err := gitOutput(ctx, dir, "symbolic-ref", "--quiet", "--short", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return ""
	}
	refRemote, branch := splitRemoteBranch(ref, remotes)
	if refRemote != remote {
		return ""
	}
	return branch
}

func remoteBranchSHA(ctx context.Context, dir, remote, branch string) (string, error) {
	return gitOutput(ctx, dir, "rev-parse", "--verify", remote+"/"+branch)
}

func aheadBehind(ctx context.Context, dir, remote, branch string) (int, int, error) {
	out, err := gitOutput(ctx, dir, "rev-list", "--left-right", "--count", "HEAD..."+remote+"/"+branch)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %s", out)
	}
	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return ahead, behind, nil
}

func sourceLastCommitAt(ctx context.Context, dir, branch string) (string, error) {
	ref := "origin/" + branch
	if _, err := gitOutput(ctx, dir, "rev-parse", "--verify", ref); err != nil {
		ref = "HEAD"
	}
	return gitOutput(ctx, dir, "show", "-s", "--format=%cI", ref)
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitRawOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
