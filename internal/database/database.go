package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"skillctl/internal/model"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	wrapped := &DB{DB: db}
	if err := wrapped.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return wrapped, nil
}

func (db *DB) init() error {
	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS sources (
			id              TEXT PRIMARY KEY,
			url             TEXT NOT NULL,
			branch          TEXT NOT NULL DEFAULT 'main',
			checkout_path   TEXT NOT NULL,
			local_sha       TEXT,
			remote_sha      TEXT,
			last_fetch_at   TEXT,
			note            TEXT,
			pinned          INTEGER NOT NULL DEFAULT 0,
			created_at      TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS skills (
			id              TEXT PRIMARY KEY,
			source_id       TEXT NOT NULL,
			relative_path   TEXT NOT NULL,
			name            TEXT NOT NULL,
			description     TEXT,
			note            TEXT,
			content_sha     TEXT,
			discovered_at   TEXT NOT NULL,
			FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
			UNIQUE(source_id, relative_path)
		)`,
		`CREATE TABLE IF NOT EXISTS skill_tags (
			skill_id        TEXT NOT NULL,
			tag             TEXT NOT NULL,
			PRIMARY KEY(skill_id, tag),
			FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS activations (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			skill_id        TEXT NOT NULL,
			agent           TEXT NOT NULL,
			scope           TEXT NOT NULL,
			project_root    TEXT,
			link_path       TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE,
			UNIQUE(link_path)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	// Migrations for databases created before a column was added. SQLite errors
	// when the column already exists, which we treat as already-migrated.
	migrations := []string{
		`ALTER TABLE sources ADD COLUMN note TEXT`,
		`ALTER TABLE sources ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE skills ADD COLUMN note TEXT`,
	}
	for _, stmt := range migrations {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (db *DB) InsertSource(src model.Source) error {
	_, err := db.Exec(`INSERT INTO sources
		(id, url, branch, checkout_path, local_sha, remote_sha, last_fetch_at, note, pinned, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		src.ID, src.URL, src.Branch, src.CheckoutPath, src.LocalSHA, src.RemoteSHA, src.LastFetchAt, src.Note, boolInt(src.Pinned), src.CreatedAt)
	return err
}

func (db *DB) UpdateSourceSHAs(id, localSHA, remoteSHA, lastFetchAt string) error {
	_, err := db.Exec(`UPDATE sources SET local_sha = ?, remote_sha = ?, last_fetch_at = ? WHERE id = ?`,
		localSHA, remoteSHA, lastFetchAt, id)
	return err
}

func (db *DB) UpdateSourceNote(id, note string) error {
	res, err := db.Exec(`UPDATE sources SET note = ? WHERE id = ?`, note, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("source %q not found", id)
	}
	return nil
}

func (db *DB) UpdateSourcePinned(id string, pinned bool) error {
	res, err := db.Exec(`UPDATE sources SET pinned = ? WHERE id = ?`, boolInt(pinned), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("source %q not found", id)
	}
	return nil
}

func (db *DB) UpdateSkillNote(id, note string) error {
	res, err := db.Exec(`UPDATE skills SET note = ? WHERE id = ?`, note, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("skill %q not found", id)
	}
	return nil
}

func (db *DB) GetSource(id string) (model.Source, error) {
	row := db.QueryRow(`SELECT id, url, branch, checkout_path, COALESCE(local_sha, ''), COALESCE(remote_sha, ''),
		COALESCE(last_fetch_at, ''), COALESCE(note, ''), COALESCE(pinned, 0), created_at FROM sources WHERE id = ?`, id)
	var src model.Source
	var pinned int
	err := row.Scan(&src.ID, &src.URL, &src.Branch, &src.CheckoutPath, &src.LocalSHA, &src.RemoteSHA, &src.LastFetchAt, &src.Note, &pinned, &src.CreatedAt)
	src.Pinned = pinned != 0
	return src, err
}

func (db *DB) ListSources() ([]model.Source, error) {
	rows, err := db.Query(`SELECT id, url, branch, checkout_path, COALESCE(local_sha, ''), COALESCE(remote_sha, ''),
		COALESCE(last_fetch_at, ''), COALESCE(note, ''), COALESCE(pinned, 0), created_at FROM sources ORDER BY pinned DESC, created_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Source
	for rows.Next() {
		var src model.Source
		var pinned int
		if err := rows.Scan(&src.ID, &src.URL, &src.Branch, &src.CheckoutPath, &src.LocalSHA, &src.RemoteSHA, &src.LastFetchAt, &src.Note, &pinned, &src.CreatedAt); err != nil {
			return nil, err
		}
		src.Pinned = pinned != 0
		out = append(out, src)
	}
	return out, rows.Err()
}

func (db *DB) CountSkillsBySource(sourceID string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM skills WHERE source_id = ?`, sourceID).Scan(&count)
	return count, err
}

func (db *DB) DeleteSource(id string) error {
	res, err := db.Exec(`DELETE FROM sources WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("source %q not found", id)
	}
	return nil
}

func (db *DB) RenameSource(oldID, newID, checkoutPath string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`PRAGMA defer_foreign_keys = ON`); err != nil {
		return err
	}

	if oldID != newID {
		var existing int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM sources WHERE id = ?`, newID).Scan(&existing); err != nil {
			return err
		}
		if existing > 0 {
			return fmt.Errorf("source %q already exists", newID)
		}
	}

	rows, err := tx.Query(`SELECT id, relative_path FROM skills WHERE source_id = ? ORDER BY id`, oldID)
	if err != nil {
		return err
	}
	type skillRef struct {
		oldID        string
		relativePath string
	}
	var refs []skillRef
	for rows.Next() {
		var ref skillRef
		if err := rows.Scan(&ref.oldID, &ref.relativePath); err != nil {
			_ = rows.Close()
			return err
		}
		refs = append(refs, ref)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	res, err := tx.Exec(`UPDATE sources SET id = ?, checkout_path = ? WHERE id = ?`, newID, checkoutPath, oldID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("source %q not found", oldID)
	}

	for _, ref := range refs {
		newSkillID := newID + "::" + ref.relativePath
		if _, err := tx.Exec(`UPDATE skill_tags SET skill_id = ? WHERE skill_id = ?`, newSkillID, ref.oldID); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE activations SET skill_id = ? WHERE skill_id = ?`, newSkillID, ref.oldID); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE skills SET id = ?, source_id = ? WHERE id = ?`, newSkillID, newID, ref.oldID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM skill_tags WHERE skill_id = ? AND tag = ?`, newSkillID, oldID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO skill_tags (skill_id, tag) VALUES (?, ?)`, newSkillID, newID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) ReplaceSkillsForSource(sourceID string, skills []model.Skill) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	seen := map[string]bool{}
	for _, skill := range skills {
		seen[skill.ID] = true
		if _, err := tx.Exec(`INSERT INTO skills
			(id, source_id, relative_path, name, description, content_sha, discovered_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			skill.ID, skill.SourceID, skill.RelativePath, skill.Name, skill.Description, skill.ContentSHA, skill.DiscoveredAt); err != nil {
			if _, updateErr := tx.Exec(`UPDATE skills SET relative_path = ?, name = ?, description = ?, content_sha = ?, discovered_at = ?
				WHERE id = ? AND source_id = ?`,
				skill.RelativePath, skill.Name, skill.Description, skill.ContentSHA, skill.DiscoveredAt, skill.ID, sourceID); updateErr != nil {
				return fmt.Errorf("%v; update failed: %w", err, updateErr)
			}
		}
	}
	rows, err := tx.Query(`SELECT id FROM skills WHERE source_id = ?`, sourceID)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		if !seen[id] {
			stale = append(stale, id)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range stale {
		if _, err := tx.Exec(`DELETE FROM skills WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListSkillsBySource(sourceID string) ([]model.Skill, error) {
	rows, err := db.Query(`SELECT id, source_id, relative_path, name, COALESCE(description, ''), COALESCE(note, ''),
		COALESCE(content_sha, ''), discovered_at FROM skills WHERE source_id = ? ORDER BY name COLLATE NOCASE, relative_path`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Skill
	for rows.Next() {
		var skill model.Skill
		if err := rows.Scan(&skill.ID, &skill.SourceID, &skill.RelativePath, &skill.Name, &skill.Description, &skill.Note, &skill.ContentSHA, &skill.DiscoveredAt); err != nil {
			return nil, err
		}
		skill.Tags, _ = db.TagsForSkill(skill.ID)
		skill.Activations, _ = db.ActivationsForSkill(skill.ID)
		out = append(out, skill)
	}
	return out, rows.Err()
}

func (db *DB) GetSkill(id string) (model.Skill, error) {
	row := db.QueryRow(`SELECT id, source_id, relative_path, name, COALESCE(description, ''), COALESCE(note, ''),
		COALESCE(content_sha, ''), discovered_at FROM skills WHERE id = ?`, id)
	var skill model.Skill
	err := row.Scan(&skill.ID, &skill.SourceID, &skill.RelativePath, &skill.Name, &skill.Description, &skill.Note, &skill.ContentSHA, &skill.DiscoveredAt)
	if err != nil {
		return skill, err
	}
	skill.Tags, _ = db.TagsForSkill(id)
	skill.Activations, _ = db.ActivationsForSkill(id)
	return skill, nil
}

func (db *DB) ListSkills() ([]model.Skill, error) {
	rows, err := db.Query(`SELECT id, source_id, relative_path, name, COALESCE(description, ''), COALESCE(note, ''),
		COALESCE(content_sha, ''), discovered_at FROM skills ORDER BY name COLLATE NOCASE, source_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Skill
	for rows.Next() {
		var skill model.Skill
		if err := rows.Scan(&skill.ID, &skill.SourceID, &skill.RelativePath, &skill.Name, &skill.Description, &skill.Note, &skill.ContentSHA, &skill.DiscoveredAt); err != nil {
			return nil, err
		}
		skill.Tags, _ = db.TagsForSkill(skill.ID)
		skill.Activations, _ = db.ActivationsForSkill(skill.ID)
		out = append(out, skill)
	}
	return out, rows.Err()
}

func (db *DB) TagsForSkill(skillID string) ([]string, error) {
	rows, err := db.Query(`SELECT tag FROM skill_tags WHERE skill_id = ? ORDER BY tag COLLATE NOCASE`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (db *DB) UpdateTags(skillIDs, tags []string, action string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, skillID := range skillIDs {
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if action == "remove" {
				if _, err := tx.Exec(`DELETE FROM skill_tags WHERE skill_id = ? AND tag = ?`, skillID, tag); err != nil {
					return err
				}
			} else {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO skill_tags (skill_id, tag) VALUES (?, ?)`, skillID, tag); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func (db *DB) InsertActivation(a model.Activation) (int64, error) {
	res, err := db.Exec(`INSERT OR IGNORE INTO activations
		(skill_id, agent, scope, project_root, link_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, a.SkillID, a.Agent, a.Scope, nullString(a.ProjectRoot), a.LinkPath, a.CreatedAt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) DeleteActivation(id int64) (model.Activation, error) {
	a, err := db.GetActivation(id)
	if err != nil {
		return a, err
	}
	_, err = db.Exec(`DELETE FROM activations WHERE id = ?`, id)
	return a, err
}

func (db *DB) DeleteActivationRecord(id int64) error {
	_, err := db.Exec(`DELETE FROM activations WHERE id = ?`, id)
	return err
}

func (db *DB) GetActivation(id int64) (model.Activation, error) {
	row := db.QueryRow(`SELECT id, skill_id, agent, scope, COALESCE(project_root, ''), link_path, created_at
		FROM activations WHERE id = ?`, id)
	var a model.Activation
	err := row.Scan(&a.ID, &a.SkillID, &a.Agent, &a.Scope, &a.ProjectRoot, &a.LinkPath, &a.CreatedAt)
	return a, err
}

func (db *DB) ListActivations() ([]model.Activation, error) {
	rows, err := db.Query(`SELECT id, skill_id, agent, scope, COALESCE(project_root, ''), link_path, created_at
		FROM activations ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivations(rows)
}

func (db *DB) ActivationsForSkill(skillID string) ([]model.Activation, error) {
	rows, err := db.Query(`SELECT id, skill_id, agent, scope, COALESCE(project_root, ''), link_path, created_at
		FROM activations WHERE skill_id = ? ORDER BY agent, scope, project_root`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivations(rows)
}

func (db *DB) ActivationsForProject(projectRoot string) ([]model.Activation, error) {
	rows, err := db.Query(`SELECT id, skill_id, agent, scope, COALESCE(project_root, ''), link_path, created_at
		FROM activations WHERE scope = 'project' AND project_root = ? ORDER BY agent, skill_id`, projectRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivations(rows)
}

func (db *DB) ActivationsForProjectAgents(projectRoot string, agents []string) ([]model.Activation, error) {
	if len(agents) == 0 {
		return db.ActivationsForProject(projectRoot)
	}
	query := `SELECT id, skill_id, agent, scope, COALESCE(project_root, ''), link_path, created_at
		FROM activations WHERE scope = 'project' AND project_root = ? AND agent IN (` + placeholders(len(agents)) + `)`
	args := []any{projectRoot}
	for _, agent := range agents {
		args = append(args, agent)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivations(rows)
}

func scanActivations(rows *sql.Rows) ([]model.Activation, error) {
	var out []model.Activation
	for rows.Next() {
		var a model.Activation
		if err := rows.Scan(&a.ID, &a.SkillID, &a.Agent, &a.Scope, &a.ProjectRoot, &a.LinkPath, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (db *DB) MustHaveSkill(skillID string) error {
	_, err := db.GetSkill(skillID)
	if err != nil {
		return fmt.Errorf("skill %q is not registered", skillID)
	}
	return nil
}
