package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"skillctl/internal/model"
)

func TestRankingCountsExplicitSkillReferencesOnly(t *testing.T) {
	home := t.TempDir()
	writeHistory(t, filepath.Join(home, ".claude", "history.jsonl"), []string{
		`{"display":"/arxiv summarize this paper","timestamp":1710000000000}`,
		`{"display":"plain arxiv mention should not count","timestamp":1710000001000}`,
		`{"display":"[$idea-discovery] use this","timestamp":1710000002000}`,
	})
	writeHistory(t, filepath.Join(home, ".codex", "history.jsonl"), []string{
		`{"text":"$arxiv compare papers","ts":1710000003}`,
		`{"text":"idea-discovery plain mention should not count","ts":1710000004}`,
	})

	manager := &Manager{
		homeDir: home,
		now:     func() time.Time { return time.Unix(1710000010, 0) },
	}
	ranking, err := manager.Ranking([]model.Skill{
		{ID: "source::skills/arxiv", SourceID: "source", RelativePath: "skills/arxiv", Name: "arxiv"},
		{ID: "source::skills/idea-discovery", SourceID: "source", RelativePath: "skills/idea-discovery", Name: "idea-discovery"},
	}, RangeAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking.Items) != 2 {
		t.Fatalf("expected 2 ranked items, got %d", len(ranking.Items))
	}
	if got := ranking.Items[0]; got.Name != "arxiv" || got.Counts.Claude != 1 || got.Counts.Codex != 1 || got.Counts.Total != 2 {
		t.Fatalf("unexpected top item: %+v", got)
	}
	if got := ranking.Items[1]; got.Name != "idea-discovery" || got.Counts.Claude != 1 || got.Counts.Codex != 0 || got.Counts.Total != 1 {
		t.Fatalf("unexpected second item: %+v", got)
	}
}

func TestRankingRangeFiltersOldEntries(t *testing.T) {
	home := t.TempDir()
	now := time.Unix(1710000000, 0)
	writeHistory(t, filepath.Join(home, ".codex", "history.jsonl"), []string{
		`{"text":"$arxiv recent","ts":1709999900}`,
		`{"text":"$arxiv old","ts":1709000000}`,
	})

	manager := &Manager{
		homeDir: home,
		now:     func() time.Time { return now },
	}
	ranking, err := manager.Ranking([]model.Skill{
		{ID: "source::skills/arxiv", SourceID: "source", RelativePath: "skills/arxiv", Name: "arxiv"},
	}, RangeDay)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking.Items) != 1 || ranking.Items[0].Counts.Total != 1 {
		t.Fatalf("expected only recent entry to count, got %+v", ranking.Items)
	}
}

func TestReadHistoryHandlesLongJSONLines(t *testing.T) {
	home := t.TempDir()
	history := filepath.Join(home, ".codex", "history.jsonl")
	longText := "$arxiv " + strings.Repeat("x", 4*1024*1024)
	writeHistory(t, history, []string{
		fmt.Sprintf(`{"text":%q,"ts":1710000003}`, longText),
	})

	entries, err := readHistory(history, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].text != longText {
		t.Fatalf("unexpected entries from long history line: count=%d", len(entries))
	}
}

func TestRankingCountsCodexExplicitAndImplicitSkillUseOncePerTurn(t *testing.T) {
	home := t.TempDir()
	rollout := filepath.Join(home, ".codex", "sessions", "2026", "06", "12", "rollout-test.jsonl")
	writeHistory(t, rollout, []string{
		rolloutEvent("2026-06-12T01:00:00Z", "event_msg", map[string]any{"type": "task_started"}),
		rolloutEvent("2026-06-12T01:00:01Z", "event_msg", map[string]any{"type": "user_message", "message": "$arxiv compare these papers"}),
		rolloutEvent("2026-06-12T01:00:02Z", "response_item", map[string]any{
			"type":      "function_call",
			"name":      "exec_command",
			"arguments": `{"cmd":"sed -n '1,220p' /tmp/skills/arxiv/SKILL.md"}`,
		}),
		rolloutEvent("2026-06-12T01:00:03Z", "event_msg", map[string]any{"type": "task_complete"}),
		rolloutEvent("2026-06-12T02:00:00Z", "event_msg", map[string]any{"type": "task_started"}),
		rolloutEvent("2026-06-12T02:00:01Z", "event_msg", map[string]any{"type": "user_message", "message": "make this prose more natural"}),
		rolloutEvent("2026-06-12T02:00:02Z", "response_item", map[string]any{
			"type":      "function_call",
			"name":      "exec_command",
			"arguments": `{"cmd":"sed -n '1,220p' /tmp/skills/humanizer/SKILL.md"}`,
		}),
		rolloutEvent("2026-06-12T02:00:03Z", "event_msg", map[string]any{"type": "task_complete"}),
	})

	manager := &Manager{
		homeDir: home,
		now:     func() time.Time { return time.Date(2026, 6, 12, 3, 0, 0, 0, time.UTC) },
	}
	ranking, err := manager.Ranking([]model.Skill{
		{ID: "source::skills/arxiv", SourceID: "source", RelativePath: "skills/arxiv", Name: "arxiv"},
		{ID: "source::skills/humanizer", SourceID: "source", RelativePath: "skills/humanizer", Name: "humanizer"},
	}, RangeDay)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking.Items) != 2 {
		t.Fatalf("expected 2 ranked items, got %+v", ranking.Items)
	}
	for _, item := range ranking.Items {
		if item.Counts.Codex != 1 || item.Counts.Total != 1 {
			t.Fatalf("expected one Codex use per turn, got %+v", item)
		}
	}
}

func TestReadCodexRolloutHandlesLongJSONLines(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex")
	rollout := filepath.Join(codexDir, "sessions", "2026", "06", "12", "rollout-long.jsonl")
	longMessage := "$arxiv " + strings.Repeat("x", 8*1024*1024)
	writeHistory(t, rollout, []string{
		rolloutEvent("2026-06-12T01:00:00Z", "event_msg", map[string]any{"type": "task_started"}),
		rolloutEvent("2026-06-12T01:00:01Z", "event_msg", map[string]any{"type": "user_message", "message": longMessage}),
		rolloutEvent("2026-06-12T01:00:02Z", "event_msg", map[string]any{"type": "task_complete"}),
	})

	entries, err := readCodexRollout(rollout, codexDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].text != longMessage {
		t.Fatalf("unexpected entries from long rollout line: count=%d", len(entries))
	}
}

func TestRankingDoesNotMultiplyAmbiguousSkillNames(t *testing.T) {
	home := t.TempDir()
	writeHistory(t, filepath.Join(home, ".codex", "history.jsonl"), []string{
		`{"text":"$arxiv compare papers","ts":1710000003}`,
	})

	manager := &Manager{
		homeDir: home,
		now:     func() time.Time { return time.Unix(1710000010, 0) },
	}
	ranking, err := manager.Ranking([]model.Skill{
		{ID: "source-a::skills/arxiv", SourceID: "source-a", RelativePath: "skills/arxiv", Name: "arxiv"},
		{ID: "source-b::skills/research/arxiv", SourceID: "source-b", RelativePath: "skills/research/arxiv", Name: "arxiv"},
	}, RangeAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking.Items) != 1 || ranking.Items[0].Counts.Total != 1 {
		t.Fatalf("expected one canonical arxiv count, got %+v", ranking.Items)
	}
}

func TestCodexRolloutCacheOnlyRescansChangedFiles(t *testing.T) {
	home := t.TempDir()
	cachePath := filepath.Join(home, "usage-cache.json")
	rollout := filepath.Join(home, ".codex", "sessions", "2026", "06", "12", "rollout-cache.jsonl")
	lines := []string{
		rolloutEvent("2026-06-12T01:00:00Z", "event_msg", map[string]any{"type": "task_started"}),
		rolloutEvent("2026-06-12T01:00:01Z", "response_item", map[string]any{
			"type":      "function_call",
			"name":      "exec_command",
			"arguments": `{"cmd":"cat /tmp/skills/arxiv/SKILL.md"}`,
		}),
		rolloutEvent("2026-06-12T01:00:02Z", "event_msg", map[string]any{"type": "task_complete"}),
	}
	writeHistory(t, rollout, lines)

	manager := &Manager{
		homeDir:   home,
		cachePath: cachePath,
		now:       func() time.Time { return time.Date(2026, 6, 12, 3, 0, 0, 0, time.UTC) },
	}
	skills := []model.Skill{
		{ID: "source::skills/arxiv", SourceID: "source", RelativePath: "skills/arxiv", Name: "arxiv"},
		{ID: "source::skills/humanizer", SourceID: "source", RelativePath: "skills/humanizer", Name: "humanizer"},
	}
	if _, err := manager.Ranking(skills, RangeAll); err != nil {
		t.Fatal(err)
	}
	firstCache := loadRolloutCache(cachePath)
	firstRollout := firstCache.Files[filepath.Base(rollout)]
	if len(firstRollout.Entries) != 1 {
		t.Fatalf("unexpected initial rollout cache: %+v", firstRollout)
	}
	if _, err := manager.Ranking(skills, RangeAll); err != nil {
		t.Fatal(err)
	}
	secondRollout := loadRolloutCache(cachePath).Files[filepath.Base(rollout)]
	if len(secondRollout.Entries) != 1 || secondRollout.ModifiedAt != firstRollout.ModifiedAt {
		t.Fatalf("unchanged rollout cache changed: before=%+v after=%+v", firstRollout, secondRollout)
	}

	lines = append(lines,
		rolloutEvent("2026-06-12T02:00:00Z", "event_msg", map[string]any{"type": "task_started"}),
		rolloutEvent("2026-06-12T02:00:01Z", "response_item", map[string]any{
			"type":      "function_call",
			"name":      "exec_command",
			"arguments": `{"cmd":"cat /tmp/skills/humanizer/SKILL.md"}`,
		}),
		rolloutEvent("2026-06-12T02:00:02Z", "event_msg", map[string]any{"type": "task_complete"}),
	)
	writeHistory(t, rollout, lines)
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(rollout, future, future); err != nil {
		t.Fatal(err)
	}
	ranking, err := manager.Ranking(skills, RangeAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranking.Items) != 2 {
		t.Fatalf("expected changed rollout to be rescanned, got %+v", ranking.Items)
	}
}

func TestRankingSnapshotPersistsAcrossManagers(t *testing.T) {
	home := t.TempDir()
	cachePath := filepath.Join(home, "usage-cache.json")
	writeHistory(t, filepath.Join(home, ".codex", "history.jsonl"), []string{
		`{"text":"$arxiv compare papers","ts":1710000003}`,
	})
	now := time.Unix(1710000010, 0)
	manager := &Manager{
		homeDir:   home,
		cachePath: cachePath,
		now:       func() time.Time { return now },
	}
	skills := []model.Skill{
		{ID: "source::skills/arxiv", SourceID: "source", RelativePath: "skills/arxiv", Name: "arxiv"},
	}
	if _, ok := manager.RankingSnapshot(RangeWeek); ok {
		t.Fatal("unexpected snapshot before refresh")
	}
	ranking, err := manager.Ranking(skills, RangeWeek)
	if err != nil {
		t.Fatal(err)
	}

	reloaded := &Manager{homeDir: home, cachePath: cachePath, now: time.Now}
	snapshot, ok := reloaded.RankingSnapshot(RangeWeek)
	if !ok {
		t.Fatal("expected persisted snapshot")
	}
	if snapshot.GeneratedAt != ranking.GeneratedAt || len(snapshot.Items) != 1 || snapshot.Items[0].Counts.Total != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if _, ok := reloaded.RankingSnapshot(RangeDay); ok {
		t.Fatal("unexpected snapshot for unrefreshed range")
	}
}

func writeHistory(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rolloutEvent(timestamp, eventType string, payload map[string]any) string {
	body, err := json.Marshal(map[string]any{
		"timestamp": timestamp,
		"type":      eventType,
		"payload":   payload,
	})
	if err != nil {
		panic(fmt.Sprintf("marshal rollout event: %v", err))
	}
	return string(body)
}
