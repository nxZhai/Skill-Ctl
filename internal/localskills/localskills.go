package localskills

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"skillctl/internal/config"
	"skillctl/internal/model"
)

const maxDepth = 6

var skipDirs = map[string]bool{
	".git":         true,
	".hg":          true,
	".svn":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
}

type Manager struct {
	Config model.Config
}

type localAgentRoot struct {
	Key  string
	Path string
}

func New(cfg model.Config) *Manager {
	return &Manager{Config: cfg}
}

func (m *Manager) List(agent string) ([]model.LocalSkill, error) {
	agents := m.agentNames(agent)
	items := []model.LocalSkill{}
	for _, name := range agents {
		roots, err := m.agentRoots(name)
		if err != nil {
			continue
		}
		for _, root := range roots {
			skills, err := scanAgentRoot(name, root)
			if err != nil {
				return nil, err
			}
			items = append(items, skills...)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Agent != items[j].Agent {
			return items[i].Agent < items[j].Agent
		}
		if items[i].AgentRoot != items[j].AgentRoot {
			return items[i].AgentRoot < items[j].AgentRoot
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (m *Manager) Content(id string) (string, error) {
	root, err := m.skillRoot(id)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(filepath.Join(root, "SKILL.md"))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (m *Manager) Tree(id string) (model.SkillTree, error) {
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

func (m *Manager) OpenDir(id, relPath string) error {
	root, err := m.skillRoot(id)
	if err != nil {
		return err
	}
	target, err := cleanSubdir(root, relPath)
	if err != nil {
		return err
	}
	return openPath(target)
}

func (m *Manager) OpenPath(id, relPath string) error {
	root, err := m.skillRoot(id)
	if err != nil {
		return err
	}
	target, err := cleanPath(root, relPath)
	if err != nil {
		return err
	}
	return openFileInPreferredEditor(target)
}

func (m *Manager) agentNames(agent string) []string {
	agent = strings.TrimSpace(agent)
	if agent != "" {
		return []string{agent}
	}
	return []string{"CLAUDE-Code", "codex"}
}

func (m *Manager) agentRoots(agent string) ([]localAgentRoot, error) {
	cfg, ok := m.Config.Agents[agent]
	if !ok && agent != "CLAUDE-Code" && agent != "codex" {
		return nil, fmt.Errorf("unknown agent %q", agent)
	}
	candidates := []string{}
	if ok {
		candidates = append(candidates, cfg.UserDir)
	}
	if agent == "CLAUDE-Code" {
		candidates = append(candidates, "~/.claude/skills")
	}
	if agent == "codex" {
		candidates = append(candidates, "~/.agents/skills", "~/.codex/skills")
	}
	roots := make([]localAgentRoot, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		root, err := config.ExpandPath(candidate)
		if err != nil || strings.TrimSpace(root) == "" {
			continue
		}
		root = filepath.Clean(root)
		if seen[root] {
			continue
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		roots = append(roots, localAgentRoot{Key: rootKey(root), Path: root})
		seen[root] = true
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no local skill directories found for %q", agent)
	}
	return roots, nil
}

func (m *Manager) skillRoot(id string) (string, error) {
	agent, rest, ok := strings.Cut(id, "::")
	rootKeyValue, rel, relOK := strings.Cut(rest, "::")
	if !ok || !relOK || strings.TrimSpace(agent) == "" || strings.TrimSpace(rootKeyValue) == "" || strings.TrimSpace(rel) == "" {
		return "", errors.New("invalid local skill id")
	}
	roots, err := m.agentRoots(agent)
	if err != nil {
		return "", err
	}
	var root string
	for _, candidate := range roots {
		if candidate.Key == rootKeyValue {
			root = candidate.Path
			break
		}
	}
	if root == "" {
		return "", errors.New("unknown local skill root")
	}
	target, err := cleanPath(root, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("local skill path is not a directory: %s", target)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		return "", err
	}
	return target, nil
}

func scanAgentRoot(agent string, root localAgentRoot) ([]model.LocalSkill, error) {
	items := []model.LocalSkill{}
	err := walkSkillFiles(root.Path, func(path string) error {
		skillDir := filepath.Dir(path)
		relDir, err := filepath.Rel(root.Path, skillDir)
		if err != nil {
			return err
		}
		relDir = filepath.ToSlash(relDir)
		meta, err := readSkill(path)
		if err != nil {
			return err
		}
		name := meta.Name
		if name == "" {
			name = filepath.Base(skillDir)
		}
		isSymlink, symlinkPath, realPath := symlinkInfo(root.Path, skillDir)
		items = append(items, model.LocalSkill{
			ID:           agent + "::" + root.Key + "::" + relDir,
			Agent:        agent,
			AgentRoot:    root.Path,
			RootKey:      root.Key,
			Root:         skillDir,
			RelativePath: relDir,
			Name:         name,
			Description:  meta.Description,
			ContentSHA:   meta.ContentSHA,
			IsSymlink:    isSymlink,
			SymlinkPath:  symlinkPath,
			RealPath:     realPath,
		})
		return nil
	})
	return items, err
}

func symlinkInfo(agentRoot, skillDir string) (bool, string, string) {
	rel, err := filepath.Rel(agentRoot, skillDir)
	if err != nil {
		return false, "", ""
	}
	current := agentRoot
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, filepath.FromSlash(part))
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		realPath, err := filepath.EvalSymlinks(current)
		if err != nil {
			realPath = ""
		}
		if realPath != "" && current != skillDir {
			suffix, err := filepath.Rel(current, skillDir)
			if err == nil && suffix != "." {
				realPath = filepath.Join(realPath, suffix)
			}
		}
		return true, current, realPath
	}
	return false, "", ""
}

func walkSkillFiles(root string, visit func(path string) error) error {
	seen := map[string]bool{}
	var walk func(dir string, depth int) error
	walk = func(dir string, depth int) error {
		if depth > maxDepth {
			return nil
		}
		realDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			if seen[realDir] {
				return nil
			}
			seen[realDir] = true
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == "SKILL.md" {
				return visit(filepath.Join(dir, entry.Name()))
			}
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				continue
			}
			if skipDirs[entry.Name()] {
				continue
			}
			if err := walk(path, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, 0)
}

func rootKey(root string) string {
	base := filepath.Base(root)
	parent := filepath.Base(filepath.Dir(root))
	key := strings.Trim(parent+"-"+base, "-")
	key = strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(key)
	if key == "" || key == "." {
		return "root"
	}
	return key
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
		entry := model.SkillTreeEntry{Name: name, Path: filepath.ToSlash(rel), Kind: "file"}
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
	return isDir && skipDirs[name]
}

func cleanSubdir(root, relPath string) (string, error) {
	target, err := cleanPath(root, relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return target, nil
}

func cleanPath(root, relPath string) (string, error) {
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
		return "", errors.New("path is outside the local skill")
	}
	if _, err := os.Stat(absTarget); err != nil {
		return "", err
	}
	return absTarget, nil
}

type skillMeta struct {
	Name        string
	Description string
	ContentSHA  string
}

func readSkill(path string) (skillMeta, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return skillMeta{}, err
	}
	sum := sha256.Sum256(body)
	meta := skillMeta{ContentSHA: hex.EncodeToString(sum[:])}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	inFrontMatter := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if lineNo == 1 && line == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter && line == "---" {
			break
		}
		if strings.HasPrefix(line, "# ") && meta.Name == "" {
			meta.Name = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		if strings.HasPrefix(line, "name:") && meta.Name == "" {
			meta.Name = cleanValue(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") && meta.Description == "" {
			meta.Description = cleanValue(strings.TrimPrefix(line, "description:"))
		}
		if !inFrontMatter && lineNo > 60 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return meta, err
	}
	return meta, nil
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func openPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
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
