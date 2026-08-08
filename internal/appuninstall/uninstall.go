// Package appuninstall removes Skillctl-managed activations and, when requested, source checkouts.
package appuninstall

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"skillctl/internal/activation"
	"skillctl/internal/database"
	"skillctl/internal/model"
	"skillctl/internal/sources"
)

// DisableAll removes every activation recorded by Skillctl. The activation manager validates each link before deleting it.
func DisableAll(db *database.DB, manager *activation.Manager) ([]model.OperationResult, error) {
	activations, err := db.ListActivations()
	if err != nil {
		return nil, err
	}
	return manager.DisableMany(activations), nil
}

// RemoveRepositories removes source records and their checkouts. All paths are verified before deletion.
func RemoveRepositories(db *database.DB, manager *sources.Manager, reposDir string) ([]string, error) {
	sourcesToRemove, err := db.ListSources()
	if err != nil {
		return nil, err
	}
	for _, source := range sourcesToRemove {
		if !inside(reposDir, source.CheckoutPath) {
			return nil, fmt.Errorf("refusing to delete source checkout outside managed repos directory: %s", source.CheckoutPath)
		}
	}
	removed := make([]string, 0, len(sourcesToRemove))
	for _, source := range sourcesToRemove {
		if err := manager.Remove(source.ID); err != nil {
			return removed, err
		}
		if err := os.RemoveAll(source.CheckoutPath); err != nil {
			return removed, err
		}
		removed = append(removed, source.CheckoutPath)
	}
	return removed, nil
}

// Confirm asks a destructive-action question and accepts only y or yes.
func Confirm(input io.Reader, output io.Writer, question string) bool {
	fmt.Fprintf(output, "%s [y/N] ", question)
	line, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && len(line) == 0 {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func inside(root, path string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
