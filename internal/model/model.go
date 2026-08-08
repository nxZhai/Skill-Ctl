package model

type Source struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Branch       string `json:"branch"`
	CheckoutPath string `json:"checkout_path"`
	LocalSHA     string `json:"local_sha,omitempty"`
	RemoteSHA    string `json:"remote_sha,omitempty"`
	LastFetchAt  string `json:"last_fetch_at,omitempty"`
	Note         string `json:"note,omitempty"`
	Pinned       bool   `json:"pinned,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type SourceView struct {
	Source
	SkillCount   int            `json:"skill_count"`
	Status       string         `json:"status"`
	Message      string         `json:"message,omitempty"`
	Behind       int            `json:"behind,omitempty"`
	Ahead        int            `json:"ahead,omitempty"`
	LastCommitAt string         `json:"last_commit_at,omitempty"`
	Remotes      []SourceRemote `json:"remotes,omitempty"`
}

type SourceRemote struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch,omitempty"`
	SHA    string `json:"sha,omitempty"`
	Ahead  int    `json:"ahead,omitempty"`
	Behind int    `json:"behind,omitempty"`
}

type Skill struct {
	ID            string       `json:"id"`
	SourceID      string       `json:"source_id"`
	RelativePath  string       `json:"relative_path"`
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	Note          string       `json:"note,omitempty"`
	ContentSHA    string       `json:"content_sha,omitempty"`
	DiscoveredAt  string       `json:"discovered_at"`
	Tags          []string     `json:"tags,omitempty"`
	Activations   []Activation `json:"activations,omitempty"`
	LocalChanged  bool         `json:"local_changed,omitempty"`
	RemoteChanged bool         `json:"remote_changed,omitempty"`
}

type LocalSkill struct {
	ID           string `json:"id"`
	Agent        string `json:"agent"`
	AgentRoot    string `json:"agent_root"`
	RootKey      string `json:"root_key"`
	Root         string `json:"root"`
	RelativePath string `json:"relative_path"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ContentSHA   string `json:"content_sha,omitempty"`
	IsSymlink    bool   `json:"is_symlink"`
	SymlinkPath  string `json:"symlink_path,omitempty"`
	RealPath     string `json:"real_path,omitempty"`
}

type SkillTree struct {
	Root    string           `json:"root"`
	Entries []SkillTreeEntry `json:"entries"`
}

type SkillTreeEntry struct {
	Name     string           `json:"name"`
	Path     string           `json:"path"`
	Kind     string           `json:"kind"`
	Children []SkillTreeEntry `json:"children,omitempty"`
}

type Activation struct {
	ID          int64  `json:"id"`
	SkillID     string `json:"skill_id"`
	Agent       string `json:"agent"`
	Scope       string `json:"scope"`
	ProjectRoot string `json:"project_root,omitempty"`
	LinkPath    string `json:"link_path"`
	CreatedAt   string `json:"created_at"`
}

type AgentConfig struct {
	UserDir    string `toml:"user_dir" json:"user_dir"`
	ProjectDir string `toml:"project_dir" json:"project_dir"`
}

type ProjectRef struct {
	Alias string `toml:"alias" json:"alias"`
	Path  string `toml:"path" json:"path"`
}

type Config struct {
	Agents    map[string]AgentConfig `toml:"agents" json:"agents"`
	Projects  []ProjectRef           `toml:"projects" json:"projects"`
	ReposDir  string                 `toml:"repos_dir" json:"repos_dir"`
	SkillsDir string                 `toml:"skills_dir" json:"skills_dir"`
}

type Paths struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	ReposDir  string
	SkillsDir string
	LocksDir  string
	LogsDir   string
	DBPath    string
}

type OperationResult struct {
	ID      string `json:"id,omitempty"`
	OK      bool   `json:"ok"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}
