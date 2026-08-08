package usage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"skillctl/internal/model"
)

type Range string

const (
	RangeDay   Range = "day"
	RangeWeek  Range = "week"
	RangeMonth Range = "month"
	RangeAll   Range = "all"
)

type Count struct {
	Claude int `json:"claude"`
	Codex  int `json:"codex"`
	Total  int `json:"total"`
}

type SkillUsage struct {
	SkillID      string `json:"skill_id"`
	SourceID     string `json:"source_id"`
	RelativePath string `json:"relative_path"`
	Name         string `json:"name"`
	Counts       Count  `json:"counts"`
}

type Summary struct {
	GeneratedAt string                `json:"generated_at"`
	Counts      map[string]SkillUsage `json:"counts"`
}

type Ranking struct {
	GeneratedAt string       `json:"generated_at"`
	Range       Range        `json:"range"`
	Items       []SkillUsage `json:"items"`
}

type Manager struct {
	homeDir   string
	cachePath string
	now       func() time.Time
	cacheMu   sync.Mutex
}

type logEntry struct {
	agent  string
	when   time.Time
	text   string
	skills []observedSkill
}

type observedSkill struct {
	name         string
	path         string
	resolvedPath string
}

type knownSkill struct {
	model.Skill
	patterns []*regexp.Regexp
	names    []string
	key      string
}

type rolloutCache struct {
	Version  int                         `json:"version"`
	Files    map[string]rolloutCacheFile `json:"files"`
	Rankings map[Range]Ranking           `json:"rankings,omitempty"`
}

type rolloutCacheFile struct {
	Size       int64            `json:"size"`
	ModifiedAt int64            `json:"modified_at"`
	Entries    []cachedLogEntry `json:"entries"`
}

type cachedLogEntry struct {
	When   int64                 `json:"when"`
	Text   string                `json:"text,omitempty"`
	Skills []cachedObservedSkill `json:"skills,omitempty"`
}

type cachedObservedSkill struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	ResolvedPath string `json:"resolved_path"`
}

func New() *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{
		homeDir:   home,
		cachePath: filepath.Join(home, ".cache", "skillctl", "usage-rollouts.json"),
		now:       time.Now,
	}
}

func (m *Manager) Summary(skills []model.Skill) (Summary, error) {
	counts, err := m.count(skills, RangeAll)
	return Summary{GeneratedAt: m.now().UTC().Format(time.RFC3339), Counts: counts}, err
}

func (m *Manager) Ranking(skills []model.Skill, r Range) (Ranking, error) {
	counts, err := m.count(skills, r)
	if err != nil {
		return Ranking{}, err
	}
	items := make([]SkillUsage, 0, len(counts))
	for _, item := range counts {
		if item.Counts.Total > 0 {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Counts.Total != items[j].Counts.Total {
			return items[i].Counts.Total > items[j].Counts.Total
		}
		if items[i].Counts.Claude != items[j].Counts.Claude {
			return items[i].Counts.Claude > items[j].Counts.Claude
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	ranking := Ranking{GeneratedAt: m.now().UTC().Format(time.RFC3339), Range: r, Items: items}
	if err := m.saveRankingSnapshot(ranking); err != nil {
		return Ranking{}, err
	}
	return ranking, nil
}

func (m *Manager) RankingSnapshot(r Range) (Ranking, bool) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	ranking, ok := loadRolloutCache(m.rolloutCachePath()).Rankings[r]
	return ranking, ok
}

func (m *Manager) count(skills []model.Skill, r Range) (map[string]SkillUsage, error) {
	known := buildKnownSkills(skills)
	counts := make(map[string]SkillUsage, len(known))
	for _, skill := range known {
		counts[skill.ID] = SkillUsage{
			SkillID:      skill.ID,
			SourceID:     skill.SourceID,
			RelativePath: skill.RelativePath,
			Name:         skill.Name,
		}
	}
	cutoff := cutoffForRange(m.now(), r)
	entries, err := m.entries(cutoff)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !cutoff.IsZero() && entry.when.Before(cutoff) {
			continue
		}
		matched := map[string]knownSkill{}
		matchedKeys := map[string]bool{}
		for _, observed := range entry.skills {
			skill, ok := matchObservedSkill(observed, known)
			if !ok || matchedKeys[skill.key] {
				continue
			}
			matched[skill.ID] = skill
			matchedKeys[skill.key] = true
		}
		for _, skill := range known {
			if matchedKeys[skill.key] || !matchesSkill(entry.text, skill.patterns) {
				continue
			}
			matched[skill.ID] = skill
			matchedKeys[skill.key] = true
		}
		for _, skill := range matched {
			item := counts[skill.ID]
			if entry.agent == "claude" {
				item.Counts.Claude++
			} else {
				item.Counts.Codex++
			}
			item.Counts.Total = item.Counts.Claude + item.Counts.Codex
			counts[skill.ID] = item
		}
	}
	return counts, nil
}

func (m *Manager) entries(cutoff time.Time) ([]logEntry, error) {
	claude, err := readHistory(filepath.Join(m.homeDir, ".claude", "history.jsonl"), "claude")
	if err != nil {
		return nil, err
	}
	codexHistory, err := readHistory(filepath.Join(m.homeDir, ".codex", "history.jsonl"), "codex")
	if err != nil {
		return nil, err
	}
	codexRollouts, err := m.readCodexRollouts(filepath.Join(m.homeDir, ".codex"), cutoff)
	if err != nil {
		return nil, err
	}
	entries := append([]logEntry{}, claude...)
	entries = append(entries, mergeCodexEntries(codexHistory, codexRollouts)...)
	return entries, nil
}

func readHistory(path, agent string) ([]logEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []logEntry
	err = scanLogLines(file, func(lineBytes []byte) error {
		line := strings.TrimSpace(string(lineBytes))
		if line == "" {
			return nil
		}
		if agent == "claude" {
			entry, ok := parseClaudeHistory(line)
			if ok {
				entries = append(entries, entry)
			}
			return nil
		}
		entry, ok := parseCodexHistory(line)
		if ok {
			entries = append(entries, entry)
		}
		return nil
	})
	return entries, err
}

func parseClaudeHistory(line string) (logEntry, bool) {
	var raw struct {
		Display   string `json:"display"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil || strings.TrimSpace(raw.Display) == "" {
		return logEntry{}, false
	}
	return logEntry{agent: "claude", when: time.UnixMilli(raw.Timestamp), text: raw.Display}, true
}

func parseCodexHistory(line string) (logEntry, bool) {
	var raw struct {
		Text string `json:"text"`
		TS   int64  `json:"ts"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil || strings.TrimSpace(raw.Text) == "" {
		return logEntry{}, false
	}
	return logEntry{agent: "codex", when: time.Unix(raw.TS, 0), text: raw.Text}, true
}

func (m *Manager) readCodexRollouts(codexDir string, cutoff time.Time) ([]logEntry, error) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	cache := loadRolloutCache(m.rolloutCachePath())
	nextCache := rolloutCache{
		Version:  2,
		Files:    map[string]rolloutCacheFile{},
		Rankings: cache.Rankings,
	}
	var entries []logEntry
	seen := map[string]bool{}
	cacheChanged := false
	for _, root := range []string{
		filepath.Join(codexDir, "sessions"),
		filepath.Join(codexDir, "archived_sessions"),
	} {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if dirEntry.IsDir() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			if seen[dirEntry.Name()] {
				return nil
			}
			seen[dirEntry.Name()] = true
			info, err := dirEntry.Info()
			if err != nil {
				return err
			}
			cached, ok := cache.Files[dirEntry.Name()]
			if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
				if ok && cached.Size == info.Size() && cached.ModifiedAt == info.ModTime().UnixNano() {
					nextCache.Files[dirEntry.Name()] = cached
				}
				return nil
			}
			if ok && cached.Size == info.Size() && cached.ModifiedAt == info.ModTime().UnixNano() {
				nextCache.Files[dirEntry.Name()] = cached
				entries = append(entries, cachedEntries(cached.Entries)...)
				return nil
			}
			next, err := readCodexRollout(path, codexDir)
			if err != nil {
				return err
			}
			nextCache.Files[dirEntry.Name()] = rolloutCacheFile{
				Size:       info.Size(),
				ModifiedAt: info.ModTime().UnixNano(),
				Entries:    cacheEntries(next),
			}
			cacheChanged = true
			entries = append(entries, next...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(nextCache.Files) != len(cache.Files) {
		cacheChanged = true
	}
	if cacheChanged {
		_ = saveRolloutCache(m.rolloutCachePath(), nextCache)
	}
	return entries, nil
}

func (m *Manager) rolloutCachePath() string {
	if m.cachePath != "" {
		return m.cachePath
	}
	return filepath.Join(m.homeDir, ".cache", "skillctl", "usage-rollouts.json")
}

func loadRolloutCache(path string) rolloutCache {
	cache := rolloutCache{
		Version:  2,
		Files:    map[string]rolloutCacheFile{},
		Rankings: map[Range]Ranking{},
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return cache
	}
	if json.Unmarshal(body, &cache) != nil || (cache.Version != 1 && cache.Version != 2) {
		cache.Version = 2
		cache.Files = map[string]rolloutCacheFile{}
		cache.Rankings = map[Range]Ranking{}
		return cache
	}
	cache.Version = 2
	if cache.Files == nil {
		cache.Files = map[string]rolloutCacheFile{}
	}
	if cache.Rankings == nil {
		cache.Rankings = map[Range]Ranking{}
	}
	return cache
}

func (m *Manager) saveRankingSnapshot(ranking Ranking) error {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	cache := loadRolloutCache(m.rolloutCachePath())
	cache.Rankings[ranking.Range] = ranking
	return saveRolloutCache(m.rolloutCachePath(), cache)
}

func saveRolloutCache(path string, cache rolloutCache) error {
	body, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func cacheEntries(entries []logEntry) []cachedLogEntry {
	out := make([]cachedLogEntry, 0, len(entries))
	for _, entry := range entries {
		cached := cachedLogEntry{
			When: entry.when.UnixNano(),
			Text: entry.text,
		}
		for _, skill := range entry.skills {
			cached.Skills = append(cached.Skills, cachedObservedSkill{
				Name:         skill.name,
				Path:         skill.path,
				ResolvedPath: skill.resolvedPath,
			})
		}
		out = append(out, cached)
	}
	return out
}

func cachedEntries(entries []cachedLogEntry) []logEntry {
	out := make([]logEntry, 0, len(entries))
	for _, cached := range entries {
		entry := logEntry{
			agent: "codex",
			when:  time.Unix(0, cached.When),
			text:  cached.Text,
		}
		for _, skill := range cached.Skills {
			entry.skills = append(entry.skills, observedSkill{
				name:         skill.Name,
				path:         skill.Path,
				resolvedPath: skill.ResolvedPath,
			})
		}
		out = append(out, entry)
	}
	return out
}

func readCodexRollout(path, codexDir string) ([]logEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []logEntry
	var current *logEntry
	flush := func() {
		if current == nil {
			return
		}
		current.text = strings.TrimSpace(current.text)
		if current.text != "" || len(current.skills) > 0 {
			entries = append(entries, *current)
		}
		current = nil
	}
	ensureCurrent := func(when time.Time) {
		if current == nil {
			current = &logEntry{agent: "codex", when: when}
		} else if current.when.IsZero() {
			current.when = when
		}
	}

	err = scanLogLines(file, func(line []byte) error {
		eventMessage := bytes.Contains(line, []byte(`"type":"event_msg"`)) &&
			(bytes.Contains(line, []byte(`"type":"task_started"`)) ||
				bytes.Contains(line, []byte(`"type":"user_message"`)) ||
				bytes.Contains(line, []byte(`"type":"task_complete"`)))
		skillCall := bytes.Contains(line, []byte(`"type":"response_item"`)) &&
			bytes.Contains(line, []byte(`"type":"function_call"`)) &&
			bytes.Contains(line, []byte("SKILL.md"))
		if !eventMessage && !skillCall {
			return nil
		}
		var event struct {
			Timestamp string          `json:"timestamp"`
			Type      string          `json:"type"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			return nil
		}
		when, _ := time.Parse(time.RFC3339Nano, event.Timestamp)
		var payloadType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(event.Payload, &payloadType); err != nil {
			return nil
		}
		switch {
		case event.Type == "event_msg" && payloadType.Type == "task_started":
			flush()
			ensureCurrent(when)
		case event.Type == "event_msg" && payloadType.Type == "user_message":
			var payload struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil {
				return nil
			}
			ensureCurrent(when)
			if strings.TrimSpace(payload.Message) != "" {
				if current.text != "" {
					current.text += "\n"
				}
				current.text += payload.Message
			}
		case event.Type == "response_item" && payloadType.Type == "function_call":
			var payload struct {
				Arguments string `json:"arguments"`
			}
			if json.Unmarshal(event.Payload, &payload) != nil {
				return nil
			}
			skills := extractObservedSkills(payload.Arguments, codexDir)
			if len(skills) == 0 {
				return nil
			}
			ensureCurrent(when)
			current.skills = appendObservedSkills(current.skills, skills...)
		case event.Type == "event_msg" && payloadType.Type == "task_complete":
			flush()
		}
		return nil
	})
	flush()
	return entries, err
}

func scanLogLines(r io.Reader, handle func([]byte) error) error {
	reader := bufio.NewReaderSize(r, 64*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if handleErr := handle(line); handleErr != nil {
				return handleErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

var skillPathPattern = regexp.MustCompile(`(?:\$HOME|~|/)[^"'\\\n\r\t;|&]*?/SKILL\.md`)

func extractObservedSkills(arguments, codexDir string) []observedSkill {
	arguments = strings.ReplaceAll(arguments, `\/`, `/`)
	homeDir := filepath.Dir(codexDir)
	var out []observedSkill
	for _, match := range skillPathPattern.FindAllString(arguments, -1) {
		path := strings.TrimSpace(match)
		switch {
		case strings.HasPrefix(path, "$HOME/"):
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "$HOME/"))
		case strings.HasPrefix(path, "~/"):
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		}
		path = filepath.Clean(path)
		resolved := path
		if next, err := filepath.EvalSymlinks(path); err == nil {
			resolved = next
		}
		out = appendObservedSkills(out, observedSkill{
			name:         filepath.Base(filepath.Dir(path)),
			path:         path,
			resolvedPath: resolved,
		})
	}
	return out
}

func appendObservedSkills(current []observedSkill, next ...observedSkill) []observedSkill {
	seen := map[string]bool{}
	for _, skill := range current {
		seen[strings.ToLower(skill.resolvedPath)] = true
	}
	for _, skill := range next {
		key := strings.ToLower(skill.resolvedPath)
		if seen[key] {
			continue
		}
		seen[key] = true
		current = append(current, skill)
	}
	return current
}

func mergeCodexEntries(history, rollouts []logEntry) []logEntry {
	rolloutByText := map[string][]time.Time{}
	for _, entry := range rollouts {
		key := normalizeText(entry.text)
		if key != "" {
			rolloutByText[key] = append(rolloutByText[key], entry.when)
		}
	}
	out := append([]logEntry{}, rollouts...)
	for _, entry := range history {
		duplicate := false
		for _, when := range rolloutByText[normalizeText(entry.text)] {
			if durationAbs(entry.when.Sub(when)) <= 5*time.Second {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, entry)
		}
	}
	return out
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func buildKnownSkills(skills []model.Skill) []knownSkill {
	skills = append([]model.Skill{}, skills...)
	sort.SliceStable(skills, func(i, j int) bool {
		leftName := strings.ToLower(skills[i].Name)
		rightName := strings.ToLower(skills[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		if len(skills[i].RelativePath) != len(skills[j].RelativePath) {
			return len(skills[i].RelativePath) < len(skills[j].RelativePath)
		}
		if skills[i].SourceID != skills[j].SourceID {
			return skills[i].SourceID < skills[j].SourceID
		}
		return skills[i].ID < skills[j].ID
	})
	out := make([]knownSkill, 0, len(skills))
	for _, skill := range skills {
		names := uniqueNames(skill.Name, filepath.Base(filepath.FromSlash(skill.RelativePath)))
		var patterns []*regexp.Regexp
		for _, name := range names {
			if len(name) < 2 {
				continue
			}
			escaped := regexp.QuoteMeta(name)
			patterns = append(patterns,
				regexp.MustCompile(`(?i)(^|[\s\[\("'，。:：,])[/\$]`+escaped+`($|[\s\]\)"'，。:：,])`),
			)
		}
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		if key == "" && len(names) > 0 {
			key = strings.ToLower(names[0])
		}
		out = append(out, knownSkill{Skill: skill, patterns: patterns, names: names, key: key})
	}
	return out
}

func uniqueNames(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func matchesSkill(text string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func matchObservedSkill(observed observedSkill, known []knownSkill) (knownSkill, bool) {
	for _, skill := range known {
		if observedPathMatches(observed, skill.RelativePath) {
			return skill, true
		}
	}
	name := strings.ToLower(strings.TrimSpace(observed.name))
	for _, skill := range known {
		for _, candidate := range skill.names {
			if strings.ToLower(candidate) == name {
				return skill, true
			}
		}
	}
	return knownSkill{}, false
}

func observedPathMatches(observed observedSkill, relativePath string) bool {
	suffix := "/" + strings.Trim(filepath.ToSlash(relativePath), "/") + "/SKILL.md"
	for _, path := range []string{observed.path, observed.resolvedPath} {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return true
		}
	}
	return false
}

func cutoffForRange(now time.Time, r Range) time.Time {
	switch r {
	case RangeDay:
		return now.Add(-24 * time.Hour)
	case RangeWeek:
		return now.Add(-7 * 24 * time.Hour)
	case RangeMonth:
		return now.Add(-30 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

func ParseRange(value string) Range {
	switch Range(value) {
	case RangeDay, RangeWeek, RangeMonth:
		return Range(value)
	default:
		return RangeAll
	}
}
