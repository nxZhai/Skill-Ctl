package project

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"skillctl/internal/activation"
	"skillctl/internal/config"
	"skillctl/internal/database"
	"skillctl/internal/model"
)

const ManifestName = ".skillctl.toml"

type Manifest struct {
	Version int `toml:"version" json:"version"`
	Project struct {
		Agents []string `toml:"agents" json:"agents"`
	} `toml:"project" json:"project"`
	Enable struct {
		Skills []string `toml:"skills" json:"skills"`
	} `toml:"enable" json:"enable"`
}

type Manager struct {
	DB          *database.DB
	Activations *activation.Manager
}

type ProjectView struct {
	Path               string             `json:"path"`
	Manifest           *Manifest          `json:"manifest,omitempty"`
	ProjectActivations []model.Activation `json:"project_activations"`
	GlobalActivations  []model.Activation `json:"global_activations"`
	MissingSkills      []string           `json:"missing_skills,omitempty"`
}

func New(db *database.DB, activations *activation.Manager) *Manager {
	return &Manager{DB: db, Activations: activations}
}

func (m *Manager) View(projectPath string) (ProjectView, error) {
	root, err := config.ExpandPath(projectPath)
	if err != nil {
		return ProjectView{}, err
	}
	projectActivations, err := m.DB.ActivationsForProject(root)
	if err != nil {
		return ProjectView{}, err
	}
	all, err := m.DB.ListActivations()
	if err != nil {
		return ProjectView{}, err
	}
	var global []model.Activation
	for _, a := range all {
		if a.Scope == "user" {
			global = append(global, a)
		}
	}
	manifest, missing, err := m.ReadManifest(root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ProjectView{}, err
	}
	return ProjectView{
		Path:               root,
		Manifest:           manifest,
		ProjectActivations: projectActivations,
		GlobalActivations:  global,
		MissingSkills:      missing,
	}, nil
}

func (m *Manager) ReadManifest(projectRoot string) (*Manifest, []string, error) {
	body, err := os.ReadFile(filepath.Join(projectRoot, ManifestName))
	if err != nil {
		return nil, nil, err
	}
	var manifest Manifest
	if err := toml.Unmarshal(body, &manifest); err != nil {
		return nil, nil, err
	}
	var missing []string
	for _, skillID := range manifest.Enable.Skills {
		if err := m.DB.MustHaveSkill(skillID); err != nil {
			missing = append(missing, skillID)
		}
	}
	return &manifest, missing, nil
}

func (m *Manager) WriteManifest(projectRoot string, agents []string) (*Manifest, error) {
	root, err := config.ExpandPath(projectRoot)
	if err != nil {
		return nil, err
	}
	activations, err := m.DB.ActivationsForProject(root)
	if err != nil {
		return nil, err
	}
	agentSet := map[string]bool{}
	skillSet := map[string]bool{}
	for _, a := range activations {
		if len(agents) > 0 && !slices.Contains(agents, a.Agent) {
			continue
		}
		agentSet[a.Agent] = true
		skillSet[a.SkillID] = true
	}
	var manifest Manifest
	manifest.Version = 1
	for agent := range agentSet {
		manifest.Project.Agents = append(manifest.Project.Agents, agent)
	}
	for skillID := range skillSet {
		manifest.Enable.Skills = append(manifest.Enable.Skills, skillID)
	}
	slices.Sort(manifest.Project.Agents)
	slices.Sort(manifest.Enable.Skills)
	body, err := toml.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, ManifestName), body, 0o644); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (m *Manager) ApplyManifest(projectRoot string) ([]model.OperationResult, error) {
	root, err := config.ExpandPath(projectRoot)
	if err != nil {
		return nil, err
	}
	manifest, missing, err := m.ReadManifest(root)
	if err != nil {
		return nil, err
	}
	var results []model.OperationResult
	for _, skillID := range missing {
		results = append(results, model.OperationResult{ID: skillID, OK: false, Status: "missing", Message: "skill is not registered locally"})
	}
	for _, agent := range manifest.Project.Agents {
		enableResults, _ := m.Activations.Enable(activation.EnableRequest{
			SkillIDs:    manifest.Enable.Skills,
			Agent:       agent,
			Scope:       "project",
			ProjectRoot: root,
		})
		for _, item := range enableResults {
			id := ""
			if item.Activation.ID != 0 {
				id = item.Activation.SkillID
			}
			results = append(results, model.OperationResult{ID: id, OK: item.OK, Message: item.Message})
		}
	}
	current, err := m.DB.ActivationsForProjectAgents(root, manifest.Project.Agents)
	if err != nil {
		return results, err
	}
	declared := map[string]bool{}
	for _, skillID := range manifest.Enable.Skills {
		declared[skillID] = true
	}
	for _, a := range current {
		if !declared[a.SkillID] {
			_, err := m.Activations.Disable(a.ID)
			results = append(results, model.OperationResult{
				ID:      a.SkillID,
				OK:      err == nil,
				Status:  "removed",
				Message: errString(err),
			})
		}
	}
	return results, nil
}

func (m *Manager) Clean(projectRoot string) ([]model.OperationResult, error) {
	root, err := config.ExpandPath(projectRoot)
	if err != nil {
		return nil, err
	}
	activations, err := m.DB.ActivationsForProject(root)
	if err != nil {
		return nil, err
	}
	return m.Activations.DisableMany(activations), nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
