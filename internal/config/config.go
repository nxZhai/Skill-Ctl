package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"skillctl/internal/model"
)

const (
	ConfigFile  = "config.toml"
	SourcesFile = "sources.toml"
)

func DefaultPaths() (model.Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return model.Paths{}, err
	}
	configDir := filepath.Join(home, ".config", "skillctl")
	dataDir := filepath.Join(home, ".local", "share", "skillctl")
	cacheDir := filepath.Join(home, ".cache", "skillctl")
	return model.Paths{
		ConfigDir: configDir,
		DataDir:   dataDir,
		CacheDir:  cacheDir,
		ReposDir:  filepath.Join(dataDir, "repos"),
		SkillsDir: filepath.Join(dataDir, "skills"),
		LocksDir:  filepath.Join(dataDir, "locks"),
		LogsDir:   filepath.Join(cacheDir, "logs"),
		DBPath:    filepath.Join(dataDir, "state.sqlite"),
	}, nil
}

func DefaultConfig(paths model.Paths) model.Config {
	return model.Config{
		ReposDir:  paths.ReposDir,
		SkillsDir: paths.SkillsDir,
		Agents: map[string]model.AgentConfig{
			"codex": {
				UserDir:    "~/.agents/skills",
				ProjectDir: ".agents/skills",
			},
			"CLAUDE-Code": {
				UserDir:    "~/.claude/skills",
				ProjectDir: ".claude/skills",
			},
		},
	}
}

func Init() (model.Paths, model.Config, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return paths, model.Config{}, err
	}
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LocksDir, paths.LogsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return paths, model.Config{}, err
		}
	}

	configPath := filepath.Join(paths.ConfigDir, ConfigFile)
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig(paths)
		body, err := toml.Marshal(cfg)
		if err != nil {
			return paths, model.Config{}, err
		}
		if err := os.WriteFile(configPath, body, 0o644); err != nil {
			return paths, model.Config{}, err
		}
	} else if err != nil {
		return paths, model.Config{}, err
	}

	sourcesPath := filepath.Join(paths.ConfigDir, SourcesFile)
	if _, err := os.Stat(sourcesPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(sourcesPath, []byte("# Sources are stored in SQLite for v0.1.\n"), 0o644); err != nil {
			return paths, model.Config{}, err
		}
	} else if err != nil {
		return paths, model.Config{}, err
	}

	cfg, err := Load(paths)
	if err != nil {
		return paths, cfg, err
	}
	paths, cfg, err = ApplyStorageDirs(paths, cfg)
	if err != nil {
		return paths, cfg, err
	}
	for _, dir := range []string{paths.ReposDir, paths.SkillsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return paths, cfg, err
		}
	}
	return paths, cfg, nil
}

func Save(paths model.Paths, cfg model.Config) error {
	body, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(paths.ConfigDir, ConfigFile), body, 0o644)
}

func Load(paths model.Paths) (model.Config, error) {
	body, err := os.ReadFile(filepath.Join(paths.ConfigDir, ConfigFile))
	if err != nil {
		return model.Config{}, err
	}
	cfg := DefaultConfig(paths)
	if err := toml.Unmarshal(body, &cfg); err != nil {
		return model.Config{}, err
	}
	if cfg.Agents == nil {
		cfg.Agents = map[string]model.AgentConfig{}
	}
	if strings.TrimSpace(cfg.ReposDir) == "" {
		cfg.ReposDir = paths.ReposDir
	}
	if strings.TrimSpace(cfg.SkillsDir) == "" {
		cfg.SkillsDir = paths.SkillsDir
	}
	normalizeDefaultAgents(cfg.Agents)
	return cfg, nil
}

func ApplyStorageDirs(paths model.Paths, cfg model.Config) (model.Paths, model.Config, error) {
	reposDir, err := ExpandPath(cfg.ReposDir)
	if err != nil {
		return paths, cfg, err
	}
	skillsDir, err := ExpandPath(cfg.SkillsDir)
	if err != nil {
		return paths, cfg, err
	}
	if strings.TrimSpace(reposDir) == "" {
		reposDir = paths.ReposDir
	}
	if strings.TrimSpace(skillsDir) == "" {
		skillsDir = paths.SkillsDir
	}
	paths.ReposDir = reposDir
	paths.SkillsDir = skillsDir
	cfg.ReposDir = reposDir
	cfg.SkillsDir = skillsDir
	return paths, cfg, nil
}

func normalizeDefaultAgents(agents map[string]model.AgentConfig) {
	if legacy, ok := agents["claude-code"]; ok {
		if _, exists := agents["CLAUDE-Code"]; !exists {
			agents["CLAUDE-Code"] = legacy
		}
		delete(agents, "claude-code")
	}
	delete(agents, "copilot")
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return filepath.Abs(path)
}
