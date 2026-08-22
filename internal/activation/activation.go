package activation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"skillctl/internal/config"
	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/scanner"
)

var safeNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Manager struct {
	DB     *database.DB
	Paths  model.Paths
	Config model.Config
}

type EnableRequest struct {
	SkillIDs    []string `json:"skill_ids"`
	Agent       string   `json:"agent"`
	Scope       string   `json:"scope"`
	ProjectRoot string   `json:"project_root,omitempty"`
}

type EnableResult struct {
	Activation model.Activation `json:"activation,omitempty"`
	OK         bool             `json:"ok"`
	Message    string           `json:"message,omitempty"`
}

func New(db *database.DB, paths model.Paths, cfg model.Config) *Manager {
	return &Manager{DB: db, Paths: paths, Config: cfg}
}

func (m *Manager) Enable(req EnableRequest) ([]EnableResult, error) {
	if len(req.SkillIDs) == 0 {
		return nil, errors.New("at least one skill is required")
	}
	if req.Scope != "user" && req.Scope != "project" {
		return nil, errors.New("scope must be user or project")
	}
	agentCfg, ok := m.Config.Agents[req.Agent]
	if !ok {
		return nil, fmt.Errorf("unknown agent %q", req.Agent)
	}
	if req.Scope == "project" && strings.TrimSpace(req.ProjectRoot) == "" {
		return nil, errors.New("project_root is required for project activations")
	}
	var results []EnableResult
	for _, skillID := range req.SkillIDs {
		a, err := m.enableOne(skillID, req.Agent, req.Scope, req.ProjectRoot, agentCfg)
		if err != nil {
			results = append(results, EnableResult{OK: false, Message: err.Error()})
			continue
		}
		results = append(results, EnableResult{OK: true, Activation: a})
	}
	return results, nil
}

func (m *Manager) enableOne(skillID, agent, scope, projectRoot string, agentCfg model.AgentConfig) (model.Activation, error) {
	skill, err := m.DB.GetSkill(skillID)
	if err != nil {
		return model.Activation{}, err
	}
	targetPath := scanner.UnifiedSkillPath(m.Paths, skill)
	if _, err := os.Stat(targetPath); err != nil {
		return model.Activation{}, fmt.Errorf("unified skill target is missing: %s", targetPath)
	}
	linkDir, normalizedProjectRoot, err := m.linkDir(scope, projectRoot, agentCfg)
	if err != nil {
		return model.Activation{}, err
	}
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		return model.Activation{}, err
	}
	linkPath := filepath.Join(linkDir, linkName(skill))
	if err := ensureManagedSymlink(linkPath, targetPath); err != nil {
		return model.Activation{}, err
	}
	a := model.Activation{
		SkillID:     skillID,
		Agent:       agent,
		Scope:       scope,
		ProjectRoot: normalizedProjectRoot,
		LinkPath:    linkPath,
		CreatedAt:   database.Now(),
	}
	id, err := m.DB.InsertActivation(a)
	if err != nil {
		return model.Activation{}, err
	}
	if id == 0 {
		// Idempotent re-enable: return the existing row if possible.
		all, _ := m.DB.ActivationsForSkill(skillID)
		for _, existing := range all {
			if existing.LinkPath == linkPath {
				return existing, nil
			}
		}
	}
	a.ID = id
	return a, nil
}

func (m *Manager) Disable(id int64) (model.Activation, error) {
	a, err := m.DB.GetActivation(id)
	if err != nil {
		return a, err
	}
	info, statErr := os.Lstat(a.LinkPath)
	if statErr == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return a, fmt.Errorf("refusing to delete non-symlink: %s", a.LinkPath)
		}
		if expected, err := m.expectedTarget(a.SkillID); err == nil {
			current, err := os.Readlink(a.LinkPath)
			if err != nil {
				return a, err
			}
			if current != expected {
				return a, fmt.Errorf("refusing to delete symlink with unexpected target: %s", a.LinkPath)
			}
		}
		if err := os.Remove(a.LinkPath); err != nil {
			return a, err
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return a, statErr
	}
	if err := m.DB.DeleteActivationRecord(id); err != nil {
		return a, err
	}
	return a, nil
}

func (m *Manager) expectedTarget(skillID string) (string, error) {
	skill, err := m.DB.GetSkill(skillID)
	if err != nil {
		return "", err
	}
	return scanner.UnifiedSkillPath(m.Paths, skill), nil
}

func (m *Manager) DisableMany(activations []model.Activation) []model.OperationResult {
	var results []model.OperationResult
	for _, a := range activations {
		_, err := m.Disable(a.ID)
		results = append(results, model.OperationResult{
			ID:      fmt.Sprintf("%d", a.ID),
			OK:      err == nil,
			Message: errString(err),
		})
	}
	return results
}

// DanglingManagedLinks returns unrecorded global activation links whose target
// is missing from Skillctl's managed skills directory.
func (m *Manager) DanglingManagedLinks() ([]string, error) {
	activations, err := m.DB.ListActivations()
	if err != nil {
		return nil, err
	}
	recorded := make(map[string]bool, len(activations))
	for _, activation := range activations {
		recorded[cleanPath(activation.LinkPath)] = true
	}
	managedRoot, err := filepath.Abs(m.Paths.SkillsDir)
	if err != nil {
		return nil, err
	}
	roots, err := m.globalLinkRoots()
	if err != nil {
		return nil, err
	}
	var dangling []string
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			linkPath := filepath.Join(root, entry.Name())
			if recorded[cleanPath(linkPath)] {
				continue
			}
			target, err := os.Readlink(linkPath)
			if err != nil {
				return nil, err
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(linkPath), target)
			}
			target, err = filepath.Abs(target)
			if err != nil || !pathWithin(managedRoot, target) {
				continue
			}
			if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
				dangling = append(dangling, linkPath)
			} else if err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(dangling)
	return dangling, nil
}

func (m *Manager) CleanupDanglingManagedLinks() ([]string, error) {
	dangling, err := m.DanglingManagedLinks()
	if err != nil {
		return nil, err
	}
	for _, path := range dangling {
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}
	return dangling, nil
}

func (m *Manager) globalLinkRoots() ([]string, error) {
	seen := map[string]bool{}
	var roots []string
	for _, agent := range m.Config.Agents {
		root, err := config.ExpandPath(agent.UserDir)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(root) == "" {
			continue
		}
		root = cleanPath(root)
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	sort.Strings(roots)
	return roots, nil
}

func cleanPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (m *Manager) linkDir(scope, projectRoot string, agentCfg model.AgentConfig) (string, string, error) {
	if scope == "user" {
		dir, err := config.ExpandPath(agentCfg.UserDir)
		return dir, "", err
	}
	root, err := config.ExpandPath(projectRoot)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(root, filepath.FromSlash(agentCfg.ProjectDir)), root, nil
}

func ensureManagedSymlink(linkPath, targetPath string) error {
	info, err := os.Lstat(linkPath)
	if errors.Is(err, os.ErrNotExist) {
		return os.Symlink(targetPath, linkPath)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("conflict: %s already exists and is not a symlink", linkPath)
	}
	current, err := os.Readlink(linkPath)
	if err != nil {
		return err
	}
	if current == targetPath {
		return nil
	}
	return fmt.Errorf("conflict: %s is a symlink to another target", linkPath)
}

func linkName(skill model.Skill) string {
	name := filepath.Base(filepath.FromSlash(scanner.UnifiedEntryRel(skill.RelativePath)))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = skill.Name
	}
	name = strings.ToLower(strings.TrimSpace(name))
	name = safeNamePattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "skill"
	}
	return name
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
