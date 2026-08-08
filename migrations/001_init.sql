CREATE TABLE IF NOT EXISTS sources (
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
);

CREATE TABLE IF NOT EXISTS skills (
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
);

CREATE TABLE IF NOT EXISTS skill_tags (
  skill_id        TEXT NOT NULL,
  tag             TEXT NOT NULL,
  PRIMARY KEY(skill_id, tag),
  FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS activations (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  skill_id        TEXT NOT NULL,
  agent           TEXT NOT NULL,
  scope           TEXT NOT NULL,
  project_root    TEXT,
  link_path       TEXT NOT NULL,
  created_at      TEXT NOT NULL,
  FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE,
  UNIQUE(link_path)
);
