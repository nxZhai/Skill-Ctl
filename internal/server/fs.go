package server

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillctl/internal/config"
)

type fsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type fsListing struct {
	Path    string    `json:"path"`
	Parent  string    `json:"parent"`
	Entries []fsEntry `json:"entries"`
}

// browseDir lists immediate sub-directories of p so the UI can pick a project
// directory. An empty path defaults to the user's home directory. Files and
// hidden directories are omitted to keep the picker focused on project roots.
func browseDir(p string) (fsListing, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fsListing{}, err
		}
		p = home
	}
	expanded, err := config.ExpandPath(p)
	if err != nil {
		return fsListing{}, err
	}
	info, err := os.Stat(expanded)
	if err != nil {
		return fsListing{}, err
	}
	if !info.IsDir() {
		expanded = filepath.Dir(expanded)
	}
	dirEntries, err := os.ReadDir(expanded)
	if err != nil {
		return fsListing{}, err
	}
	entries := make([]fsEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entries = append(entries, fsEntry{Name: name, Path: filepath.Join(expanded, name)})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	parent := filepath.Dir(expanded)
	if parent == expanded {
		parent = ""
	}
	return fsListing{Path: expanded, Parent: parent, Entries: entries}, nil
}
