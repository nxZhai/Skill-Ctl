package scanner

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillctl/internal/database"
	"skillctl/internal/model"
)

const MaxDepth = 6

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}

type Scanner struct {
	DB    *database.DB
	Paths model.Paths
}

func New(db *database.DB, paths model.Paths) *Scanner {
	return &Scanner{DB: db, Paths: paths}
}

func (s *Scanner) ScanSource(source model.Source) ([]model.Skill, error) {
	var skills []model.Skill
	root := source.CheckoutPath
	walkRoots, err := skillSearchRoots(root)
	if err != nil {
		return nil, err
	}
	if usesDedicatedSkillRoots(root, walkRoots) {
		if err := appendRootSkill(root, source, &skills); err != nil {
			return nil, err
		}
	}
	for _, walkRoot := range walkRoots {
		if err := scanSkillRoot(root, walkRoot, source, &skills); err != nil {
			return nil, err
		}
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].ID < skills[j].ID
	})
	if err := s.DB.ReplaceSkillsForSource(source.ID, skills); err != nil {
		return nil, err
	}
	if len(skills) > 0 {
		ids := make([]string, 0, len(skills))
		for _, skill := range skills {
			ids = append(ids, skill.ID)
		}
		if err := s.DB.UpdateTags(ids, []string{source.ID}, "add"); err != nil {
			return nil, err
		}
	}
	if err := s.RebuildUnifiedLinks(source, skills); err != nil {
		return nil, err
	}
	return skills, nil
}

func skillSearchRoots(root string) ([]string, error) {
	var roots []string
	conventionalRoot := filepath.Join(root, "skills")
	if info, err := os.Stat(conventionalRoot); err == nil && info.IsDir() {
		roots = append(roots, conventionalRoot)
	}
	if entries, err := os.ReadDir(root); err == nil {
		roots = appendNestedSkillRoots(roots, root, entries)
	}
	if len(roots) == 0 {
		pluginRoot := filepath.Join(root, "plugins")
		if entries, err := os.ReadDir(pluginRoot); err == nil {
			roots = appendNestedSkillRoots(roots, pluginRoot, entries)
		}
	}
	if len(roots) > 0 {
		return roots, nil
	}
	return []string{root}, nil
}

func usesDedicatedSkillRoots(root string, walkRoots []string) bool {
	return len(walkRoots) != 1 || filepath.Clean(walkRoots[0]) != filepath.Clean(root)
}

func appendNestedSkillRoots(roots []string, parent string, entries []fs.DirEntry) []string {
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "skills" || skipDirs[entry.Name()] {
			continue
		}
		candidate := filepath.Join(parent, entry.Name(), "skills")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			roots = append(roots, candidate)
		}
	}
	return roots
}

func appendRootSkill(repoRoot string, source model.Source, skills *[]model.Skill) error {
	skillFile := filepath.Join(repoRoot, "SKILL.md")
	if info, err := os.Stat(skillFile); err != nil || info.IsDir() {
		return nil
	}
	return appendSkill(repoRoot, repoRoot, skillFile, source, skills)
}

func scanSkillRoot(repoRoot, walkRoot string, source model.Source, skills *[]model.Skill) error {
	return filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == walkRoot {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		depth := len(strings.Split(filepath.ToSlash(rel), "/"))
		if d.IsDir() {
			if skipDirs[d.Name()] || depth > MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "SKILL.md" {
			return nil
		}
		skillDir := filepath.Dir(path)
		return appendSkill(repoRoot, skillDir, path, source, skills)
	})
}

func appendSkill(repoRoot, skillDir, skillFile string, source model.Source, skills *[]model.Skill) error {
	relDir, err := filepath.Rel(repoRoot, skillDir)
	if err != nil {
		return err
	}
	relDir = filepath.ToSlash(relDir)
	meta, err := readSkill(skillFile)
	if err != nil {
		return err
	}
	name := meta.Name
	if name == "" {
		name = filepath.Base(skillDir)
	}
	*skills = append(*skills, model.Skill{
		ID:           source.ID + "::" + relDir,
		SourceID:     source.ID,
		RelativePath: relDir,
		Name:         name,
		Description:  meta.Description,
		ContentSHA:   meta.ContentSHA,
		DiscoveredAt: database.Now(),
	})
	return nil
}

func (s *Scanner) RebuildUnifiedLinks(source model.Source, skills []model.Skill) error {
	sourceEntryRoot := filepath.Join(s.Paths.SkillsDir, source.ID)
	if err := os.MkdirAll(sourceEntryRoot, 0o755); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, skill := range skills {
		entryRel := UnifiedEntryRel(skill.RelativePath)
		seen[entryRel] = true
		linkPath := filepath.Join(sourceEntryRoot, filepath.FromSlash(entryRel))
		targetPath := filepath.Join(source.CheckoutPath, filepath.FromSlash(skill.RelativePath))
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return err
		}
		if err := ensureSymlink(linkPath, targetPath); err != nil {
			return err
		}
	}
	return filepath.WalkDir(sourceEntryRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == sourceEntryRoot || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceEntryRoot, path)
		if err != nil {
			return err
		}
		if !seen[filepath.ToSlash(rel)] {
			if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return os.Remove(path)
			}
		}
		return nil
	})
}

func UnifiedEntryRel(relativePath string) string {
	rel := filepath.ToSlash(relativePath)
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || rel == "." {
		return "_root"
	}
	if strings.HasPrefix(rel, "skills/") {
		return strings.TrimPrefix(rel, "skills/")
	}
	return rel
}

func UnifiedSkillPath(paths model.Paths, skill model.Skill) string {
	return filepath.Join(paths.SkillsDir, skill.SourceID, filepath.FromSlash(UnifiedEntryRel(skill.RelativePath)))
}

func ensureSymlink(linkPath, targetPath string) error {
	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return errors.New("unified skill entry exists and is not a symlink: " + linkPath)
		}
		current, err := os.Readlink(linkPath)
		if err != nil {
			return err
		}
		if current == targetPath {
			return nil
		}
		if err := os.Remove(linkPath); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Symlink(targetPath, linkPath)
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
