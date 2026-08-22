package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"skillctl/internal/activation"
	"skillctl/internal/config"
	"skillctl/internal/database"
	"skillctl/internal/doctor"
	"skillctl/internal/localskills"
	"skillctl/internal/model"
	"skillctl/internal/project"
	"skillctl/internal/sources"
	"skillctl/internal/usage"
	"skillctl/web"
)

type Server struct {
	DB          *database.DB
	Config      model.Config
	Paths       model.Paths
	Token       string
	Sources     *sources.Manager
	LocalSkills *localskills.Manager
	Activations *activation.Manager
	Projects    *project.Manager
	Doctor      *doctor.Doctor
	Usage       *usage.Manager
}

const sourceNoteWordLimit = 50

func New(db *database.DB, paths model.Paths, cfg model.Config, token string) *Server {
	activations := activation.New(db, paths, cfg)
	return &Server{
		DB:          db,
		Config:      cfg,
		Paths:       paths,
		Token:       token,
		Sources:     sources.New(db, paths),
		LocalSkills: localskills.New(cfg),
		Activations: activations,
		Projects:    project.New(db, activations),
		Doctor:      doctor.New(db, paths, cfg),
		Usage:       usage.New(),
	}
}

func (s *Server) ListenAndServe(ctx context.Context) (string, error) {
	return s.listenAndServe(ctx, s.routes(), "/?token=")
}

// ListenAndServeHeadless starts the token-protected API without serving the
// embedded frontend. It is intended for CLI and automation workflows.
func (s *Server) ListenAndServeHeadless(ctx context.Context) (string, error) {
	return s.listenAndServe(ctx, s.apiRoutes(), "/api?token=")
}

func (s *Server) listenAndServe(ctx context.Context, handler http.Handler, urlSuffix string) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	url := "http://" + listener.Addr().String() + urlSuffix + s.Token
	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	go func() {
		_ = httpServer.Serve(listener)
	}()
	return url, nil
}

func (s *Server) routes() http.Handler {
	mux := s.apiRoutes()
	mux.HandleFunc("/", s.handleStatic)
	return mux
}

func (s *Server) apiRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.withToken(s.handleAPI))
	return mux
}

func (s *Server) withToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("X-Skillctl-Token")
		}
		if token == "" && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		if token != s.Token {
			writeError(w, http.StatusUnauthorized, "invalid or missing token")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	switch {
	case path == "/config" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"agents":     s.Config.Agents,
			"projects":   s.Config.Projects,
			"repos_dir":  s.Config.ReposDir,
			"skills_dir": s.Config.SkillsDir,
		})
	case path == "/config" && r.Method == http.MethodPost:
		var req struct {
			Agents    map[string]model.AgentConfig `json:"agents"`
			Projects  []model.ProjectRef           `json:"projects"`
			ReposDir  string                       `json:"repos_dir"`
			SkillsDir string                       `json:"skills_dir"`
		}
		if !decode(w, r, &req) {
			return
		}
		if req.Agents == nil {
			req.Agents = map[string]model.AgentConfig{}
		}
		projects := make([]model.ProjectRef, 0, len(req.Projects))
		for _, p := range req.Projects {
			p.Alias = strings.TrimSpace(p.Alias)
			p.Path = strings.TrimSpace(p.Path)
			if p.Alias == "" || p.Path == "" {
				continue
			}
			projects = append(projects, p)
		}
		newCfg := model.Config{Agents: req.Agents, Projects: projects, ReposDir: strings.TrimSpace(req.ReposDir), SkillsDir: strings.TrimSpace(req.SkillsDir)}
		newPaths, newCfg, err := config.ApplyStorageDirs(s.Paths, newCfg)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		for _, dir := range []string{newPaths.ReposDir, newPaths.SkillsDir} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if err := config.Save(s.Paths, newCfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.Paths = newPaths
		s.Config = newCfg
		s.Sources.Paths = newPaths
		s.Sources.Scanner.Paths = newPaths
		s.LocalSkills.Config = newCfg
		s.Activations.Paths = newPaths
		s.Activations.Config = newCfg
		s.Doctor.Paths = newPaths
		s.Doctor.Config = newCfg
		writeJSON(w, http.StatusOK, map[string]any{
			"agents":     newCfg.Agents,
			"projects":   newCfg.Projects,
			"repos_dir":  newCfg.ReposDir,
			"skills_dir": newCfg.SkillsDir,
		})
	case path == "/fs" && r.Method == http.MethodGet:
		listing, err := browseDir(r.URL.Query().Get("path"))
		writeJSONOrError(w, listing, err)
	case path == "/sources" && r.Method == http.MethodGet:
		items, err := s.Sources.List(r.Context())
		writeJSONOrError(w, items, err)
	case path == "/sources" && r.Method == http.MethodPost:
		var req struct {
			ID     string `json:"id"`
			URL    string `json:"url"`
			Branch string `json:"branch"`
		}
		if !decode(w, r, &req) {
			return
		}
		source, skills, err := s.Sources.Add(r.Context(), req.ID, req.URL, req.Branch)
		writeJSONOrError(w, map[string]any{"source": source, "skills": skills}, err)
	case path == "/sources/check-all" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusOK, s.Sources.CheckAll(r.Context()))
	case path == "/sources/sync-all" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusOK, s.Sources.SyncAll(r.Context()))
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/check") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/check")
		id, _ = url.PathUnescape(id)
		view, err := s.Sources.Check(r.Context(), id)
		writeJSONOrError(w, view, err)
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/sync") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/sync")
		id, _ = url.PathUnescape(id)
		view, skills, err := s.Sources.Sync(r.Context(), id)
		writeJSONOrError(w, map[string]any{"source": view, "skills": skills}, err)
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/note") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/note")
		id, _ = url.PathUnescape(id)
		var req struct {
			Note string `json:"note"`
		}
		if !decode(w, r, &req) {
			return
		}
		note, err := normalizeSourceNote(req.Note)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = s.DB.UpdateSourceNote(id, note)
		writeJSONOrError(w, map[string]any{"ok": err == nil}, err)
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/pin") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/pin")
		id, _ = url.PathUnescape(id)
		var req struct {
			Pinned bool `json:"pinned"`
		}
		if !decode(w, r, &req) {
			return
		}
		if err := s.DB.UpdateSourcePinned(id, req.Pinned); err != nil {
			writeJSONOrError(w, nil, err)
			return
		}
		src, err := s.DB.GetSource(id)
		if err != nil {
			writeJSONOrError(w, nil, err)
			return
		}
		view, err := s.Sources.View(r.Context(), src)
		writeJSONOrError(w, view, err)
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/rename") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/rename")
		id, _ = url.PathUnescape(id)
		var req struct {
			ID string `json:"id"`
		}
		if !decode(w, r, &req) {
			return
		}
		view, skills, err := s.Sources.Rename(r.Context(), id, req.ID)
		writeJSONOrError(w, map[string]any{"source": view, "skills": skills}, err)
	case strings.HasPrefix(path, "/sources/") && strings.HasSuffix(path, "/open-dir") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/sources/"), "/open-dir")
		id, _ = url.PathUnescape(id)
		err := s.Sources.OpenSourceDir(id)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case strings.HasPrefix(path, "/sources/") && r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "/sources/")
		id, _ = url.PathUnescape(id)
		err := s.Sources.Remove(id)
		writeJSONOrError(w, map[string]bool{"ok": err == nil}, err)
	case path == "/skills" && r.Method == http.MethodGet:
		items, err := s.DB.ListSkills()
		if err == nil {
			items = s.Sources.AnnotateSkillChanges(r.Context(), items)
		}
		writeJSONOrError(w, items, err)
	case path == "/usage/summary" && r.Method == http.MethodGet:
		skills, err := s.DB.ListSkills()
		if err != nil {
			writeJSONOrError(w, nil, err)
			return
		}
		summary, err := s.Usage.Summary(skills)
		writeJSONOrError(w, summary, err)
	case path == "/usage/ranking" && r.Method == http.MethodGet:
		skills, err := s.DB.ListSkills()
		if err != nil {
			writeJSONOrError(w, nil, err)
			return
		}
		ranking, err := s.Usage.Ranking(skills, usage.ParseRange(r.URL.Query().Get("range")))
		writeJSONOrError(w, ranking, err)
	case path == "/usage/ranking-snapshot" && r.Method == http.MethodGet:
		ranking, ok := s.Usage.RankingSnapshot(usage.ParseRange(r.URL.Query().Get("range")))
		writeJSONOrError(w, map[string]any{"ranking": ranking, "available": ok}, nil)
	case path == "/local-skills" && r.Method == http.MethodGet:
		items, err := s.LocalSkills.List(r.URL.Query().Get("agent"))
		writeJSONOrError(w, items, err)
	case strings.HasPrefix(path, "/local-skills/") && strings.HasSuffix(path, "/content") && r.Method == http.MethodGet:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/local-skills/"), "/content")
		id, _ = url.PathUnescape(id)
		content, err := s.LocalSkills.Content(id)
		writeJSONOrError(w, map[string]string{"content": content}, err)
	case strings.HasPrefix(path, "/local-skills/") && strings.HasSuffix(path, "/tree") && r.Method == http.MethodGet:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/local-skills/"), "/tree")
		id, _ = url.PathUnescape(id)
		tree, err := s.LocalSkills.Tree(id)
		writeJSONOrError(w, tree, err)
	case strings.HasPrefix(path, "/local-skills/") && strings.HasSuffix(path, "/open-dir") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/local-skills/"), "/open-dir")
		id, _ = url.PathUnescape(id)
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		err := s.LocalSkills.OpenDir(id, req.Path)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case strings.HasPrefix(path, "/local-skills/") && strings.HasSuffix(path, "/open-path") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/local-skills/"), "/open-path")
		id, _ = url.PathUnescape(id)
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		err := s.LocalSkills.OpenPath(id, req.Path)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case strings.HasPrefix(path, "/skills/") && strings.HasSuffix(path, "/content") && r.Method == http.MethodGet:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/skills/"), "/content")
		id, _ = url.PathUnescape(id)
		content, err := s.Sources.SkillContent(id)
		writeJSONOrError(w, map[string]string{"content": content}, err)
	case strings.HasPrefix(path, "/skills/") && strings.HasSuffix(path, "/tree") && r.Method == http.MethodGet:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/skills/"), "/tree")
		id, _ = url.PathUnescape(id)
		tree, err := s.Sources.SkillTree(id)
		writeJSONOrError(w, tree, err)
	case strings.HasPrefix(path, "/skills/") && strings.HasSuffix(path, "/open-dir") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/skills/"), "/open-dir")
		id, _ = url.PathUnescape(id)
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		err := s.Sources.OpenSkillDir(id, req.Path)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case strings.HasPrefix(path, "/skills/") && strings.HasSuffix(path, "/open-path") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/skills/"), "/open-path")
		id, _ = url.PathUnescape(id)
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		err := s.Sources.OpenSkillPath(id, req.Path)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case strings.HasPrefix(path, "/skills/") && strings.HasSuffix(path, "/note") && r.Method == http.MethodPost:
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/skills/"), "/note")
		id, _ = url.PathUnescape(id)
		var req struct {
			Note string `json:"note"`
		}
		if !decode(w, r, &req) {
			return
		}
		note, err := normalizeSkillNote(req.Note)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		err = s.DB.UpdateSkillNote(id, note)
		writeJSONOrError(w, map[string]any{"ok": err == nil}, err)
	case strings.HasPrefix(path, "/skills/") && r.Method == http.MethodGet:
		id := strings.TrimPrefix(path, "/skills/")
		id, _ = url.PathUnescape(id)
		item, err := s.DB.GetSkill(id)
		writeJSONOrError(w, item, err)
	case path == "/skills/tags" && r.Method == http.MethodPost:
		var req struct {
			SkillIDs []string `json:"skill_ids"`
			Tags     []string `json:"tags"`
			Action   string   `json:"action"`
		}
		if !decode(w, r, &req) {
			return
		}
		if req.Action == "" {
			req.Action = "add"
		}
		err := s.DB.UpdateTags(req.SkillIDs, req.Tags, req.Action)
		writeJSONOrError(w, map[string]bool{"ok": true}, err)
	case path == "/activations" && r.Method == http.MethodGet:
		items, err := s.DB.ListActivations()
		writeJSONOrError(w, items, err)
	case path == "/activations" && r.Method == http.MethodPost:
		var req activation.EnableRequest
		if !decode(w, r, &req) {
			return
		}
		results, err := s.Activations.Enable(req)
		writeJSONOrError(w, results, err)
	case strings.HasPrefix(path, "/activations/") && r.Method == http.MethodDelete:
		idText := strings.TrimPrefix(path, "/activations/")
		id, err := strconv.ParseInt(idText, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid activation id")
			return
		}
		a, err := s.Activations.Disable(id)
		writeJSONOrError(w, a, err)
	case path == "/project" && r.Method == http.MethodGet:
		view, err := s.Projects.View(r.URL.Query().Get("path"))
		writeJSONOrError(w, view, err)
	case path == "/project/apply" && r.Method == http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		results, err := s.Projects.ApplyManifest(req.Path)
		writeJSONOrError(w, results, err)
	case path == "/project/write-manifest" && r.Method == http.MethodPost:
		var req struct {
			Path   string   `json:"path"`
			Agents []string `json:"agents"`
		}
		if !decode(w, r, &req) {
			return
		}
		manifest, err := s.Projects.WriteManifest(req.Path, req.Agents)
		writeJSONOrError(w, manifest, err)
	case path == "/project/clean" && r.Method == http.MethodPost:
		var req struct {
			Path string `json:"path"`
		}
		if !decode(w, r, &req) {
			return
		}
		results, err := s.Projects.Clean(req.Path)
		writeJSONOrError(w, results, err)
	case path == "/doctor" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, s.Doctor.Run(r.Context(), r.URL.Query().Get("project_path")))
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func normalizeSourceNote(note string) (string, error) {
	return normalizeNote(note, "source")
}

func normalizeSkillNote(note string) (string, error) {
	return normalizeNote(note, "skill")
}

func normalizeNote(note, subject string) (string, error) {
	note = strings.Join(strings.Fields(note), " ")
	if countSourceNoteWords(note) > sourceNoteWordLimit {
		return "", fmt.Errorf("%s note must be at most %d words", subject, sourceNoteWordLimit)
	}
	return note, nil
}

func countSourceNoteWords(note string) int {
	count := 0
	inWord := false
	for _, r := range note {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			count++
			inWord = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
			}
			inWord = true
			continue
		}
		inWord = false
	}
	return count
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || path == "index.html" {
		serveIndex(w, sub)
		return
	}
	info, err := fs.Stat(sub, path)
	if err != nil || info.IsDir() {
		serveIndex(w, sub)
		return
	}
	r.URL.Path = "/" + path
	http.FileServer(http.FS(sub)).ServeHTTP(w, r)
}

func serveIndex(w http.ResponseWriter, dist fs.FS) {
	content, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func writeJSONOrError(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
