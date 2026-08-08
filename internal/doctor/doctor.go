package doctor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/project"
	"skillctl/internal/scanner"
)

type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Path    string `json:"path,omitempty"`
}

type Doctor struct {
	DB    *database.DB
	Paths model.Paths
}

func New(db *database.DB, paths model.Paths) *Doctor {
	return &Doctor{DB: db, Paths: paths}
}

func (d *Doctor) Run(ctx context.Context, projectPath string) []Check {
	var checks []Check
	if _, err := exec.LookPath("git"); err != nil {
		checks = append(checks, Check{Name: "git installed", OK: false, Message: err.Error()})
	} else {
		checks = append(checks, Check{Name: "git installed", OK: true})
	}
	sources, err := d.DB.ListSources()
	if err != nil {
		return append(checks, Check{Name: "database", OK: false, Message: err.Error()})
	}
	for _, src := range sources {
		if _, err := os.Stat(src.CheckoutPath); err != nil {
			checks = append(checks, Check{Name: "source clone exists", OK: false, Path: src.CheckoutPath, Message: err.Error()})
			continue
		}
		checks = append(checks, Check{Name: "source clone exists", OK: true, Path: src.CheckoutPath})
		if changed, err := sourceChanged(ctx, src.CheckoutPath); err != nil {
			checks = append(checks, Check{Name: "source local changes", OK: false, Path: src.CheckoutPath, Message: err.Error()})
		} else if changed {
			checks = append(checks, Check{Name: "source local changes", OK: false, Path: src.CheckoutPath, Message: "repository has uncommitted changes"})
		} else {
			checks = append(checks, Check{Name: "source local changes", OK: true, Path: src.CheckoutPath})
		}
	}
	skills, err := d.DB.ListSkills()
	if err != nil {
		checks = append(checks, Check{Name: "skills query", OK: false, Message: err.Error()})
	} else {
		for _, skill := range skills {
			link := scanner.UnifiedSkillPath(d.Paths, skill)
			if err := checkSymlinkTarget(link); err != nil {
				checks = append(checks, Check{Name: "unified skill link", OK: false, Path: link, Message: err.Error()})
			}
		}
	}
	activations, err := d.DB.ListActivations()
	if err != nil {
		checks = append(checks, Check{Name: "activation query", OK: false, Message: err.Error()})
	} else {
		for _, a := range activations {
			if err := checkSymlinkTarget(a.LinkPath); err != nil {
				checks = append(checks, Check{Name: "activation link", OK: false, Path: a.LinkPath, Message: err.Error()})
			} else {
				checks = append(checks, Check{Name: "activation link", OK: true, Path: a.LinkPath})
			}
		}
	}
	if strings.TrimSpace(projectPath) != "" {
		checks = append(checks, d.checkManifest(projectPath)...)
	}
	return checks
}

func (d *Doctor) checkManifest(projectPath string) []Check {
	root, err := filepath.Abs(expandHome(projectPath))
	if err != nil {
		return []Check{{Name: "project manifest", OK: false, Message: err.Error()}}
	}
	body, err := os.ReadFile(filepath.Join(root, project.ManifestName))
	if err != nil {
		return []Check{{Name: "project manifest", OK: false, Path: filepath.Join(root, project.ManifestName), Message: err.Error()}}
	}
	var manifest project.Manifest
	if err := toml.Unmarshal(body, &manifest); err != nil {
		return []Check{{Name: "project manifest", OK: false, Path: filepath.Join(root, project.ManifestName), Message: err.Error()}}
	}
	var checks []Check
	for _, skillID := range manifest.Enable.Skills {
		if err := d.DB.MustHaveSkill(skillID); err != nil {
			checks = append(checks, Check{Name: "manifest skill", OK: false, Message: err.Error()})
		} else {
			checks = append(checks, Check{Name: "manifest skill", OK: true, Message: skillID})
		}
	}
	return checks
}

func checkSymlinkTarget(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return os.ErrInvalid
	}
	target, err := os.Readlink(path)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	_, err = os.Stat(target)
	return err
}

func sourceChanged(ctx context.Context, dir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
