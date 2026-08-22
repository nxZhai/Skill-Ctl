package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"skillctl/internal/activation"
	"skillctl/internal/appuninstall"
	"skillctl/internal/appupdate"
	"skillctl/internal/config"
	"skillctl/internal/database"
	"skillctl/internal/doctor"
	"skillctl/internal/scanner"
	"skillctl/internal/server"
	"skillctl/internal/sources"
)

const version = "0.6.2"

func main() {
	cmd := "ui"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "ui":
		if err := runUI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "headless":
		if err := runHeadless(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "doctor":
		if err := runDoctor(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "rescan":
		only := ""
		if len(os.Args) > 2 {
			only = os.Args[2]
		}
		if err := runRescan(only); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "update":
		if err := runUpdate(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "uninstall":
		if err := runUninstall(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println("skillctl", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\nUsage:\n  skillctl ui\n  skillctl headless\n  skillctl doctor\n  skillctl rescan [source-id]\n  skillctl update [--check]\n  skillctl uninstall\n  skillctl version\n", cmd)
		os.Exit(2)
	}
}

func runUI() error {
	paths, cfg, err := config.Init()
	if err != nil {
		return err
	}
	db, err := database.Open(paths.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token, err := randomToken()
	if err != nil {
		return err
	}
	srv := server.New(db, paths, cfg, token)
	url, err := srv.ListenAndServe(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Skillctl UI:", url)
	notifyUpdateAvailable()
	_ = exec.Command("open", url).Start()
	<-ctx.Done()
	return nil
}

func runHeadless() error {
	paths, cfg, err := config.Init()
	if err != nil {
		return err
	}
	db, err := database.Open(paths.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token, err := randomToken()
	if err != nil {
		return err
	}
	srv := server.New(db, paths, cfg, token)
	url, err := srv.ListenAndServeHeadless(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Skillctl headless API:", url)
	<-ctx.Done()
	return nil
}

func notifyUpdateAvailable() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		release, newer, err := appupdate.Check(ctx, nil, version)
		if err == nil && newer {
			fmt.Printf("Skillctl update available: %s (run: skillctl update)\n", release.TagName)
		}
	}()
}

func runUpdate(args []string) error {
	checkOnly := false
	for _, arg := range args {
		if arg == "--check" {
			checkOnly = true
			continue
		}
		return fmt.Errorf("unknown update option %q", arg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	release, newer, err := appupdate.Check(ctx, nil, version)
	if err != nil {
		return err
	}
	if !newer {
		fmt.Printf("Skillctl %s is up to date.\n", version)
		return nil
	}
	if checkOnly {
		fmt.Printf("Update available: %s (installed: %s). Run `skillctl update` to install it.\n", release.TagName, version)
		return nil
	}
	paths, _, err := config.Init()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	result, err := appupdate.Apply(ctx, nil, paths, version, executable, release)
	if err != nil {
		if result.BackupPath != "" {
			return fmt.Errorf("%w (backup: %s)", err, result.BackupPath)
		}
		return err
	}
	fmt.Printf("Updated Skillctl to %s. User data was unchanged. Backup: %s\n", result.ReleaseTag, result.BackupPath)
	return nil
}

func runUninstall() error {
	paths, cfg, err := config.Init()
	if err != nil {
		return err
	}
	db, err := database.Open(paths.DBPath)
	if err != nil {
		return err
	}
	activationManager := activation.New(db, paths, cfg)
	results, err := appuninstall.DisableAll(db, activationManager)
	if err != nil {
		db.Close()
		return err
	}
	failed := false
	for _, result := range results {
		if !result.OK {
			failed = true
			fmt.Fprintf(os.Stderr, "Could not remove managed link %s: %s\n", result.ID, result.Message)
		}
	}
	if failed {
		db.Close()
		return errors.New("uninstall stopped because one or more managed links could not be removed")
	}
	fmt.Printf("Removed %d managed agent skill link(s).\n", len(results))

	removeRepos := appuninstall.Confirm(os.Stdin, os.Stdout, "Also delete Skillctl-managed local skill repositories?")
	if removeRepos {
		removed, err := appuninstall.RemoveRepositories(db, sources.New(db, paths), paths.ReposDir)
		if err != nil {
			db.Close()
			return err
		}
		fmt.Printf("Deleted %d managed local skill repository/repositories.\n", len(removed))
	} else {
		fmt.Println("Kept local skill repositories and Skillctl state for a possible future reinstall.")
	}
	if err := db.Close(); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.Remove(executable); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove installed binary %s: %w", executable, err)
	}
	fmt.Printf("Removed Skillctl binary: %s\n", executable)
	return nil
}

func runDoctor() error {
	paths, cfg, err := config.Init()
	if err != nil {
		return err
	}
	db, err := database.Open(paths.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	checks := doctor.New(db, paths, cfg).Run(context.Background(), "")
	failed := false
	for _, check := range checks {
		state := "ok"
		if !check.OK {
			state = "fail"
			failed = true
		}
		parts := []string{state, check.Name}
		if check.Path != "" {
			parts = append(parts, check.Path)
		}
		if check.Message != "" {
			parts = append(parts, check.Message)
		}
		fmt.Println(strings.Join(parts, " | "))
	}
	if failed {
		return fmt.Errorf("doctor found issues")
	}
	return nil
}

// runRescan re-scans existing checkouts and rewrites the skill set per source.
// Stale skills (e.g. previously discovered under plugins/) are removed via
// ReplaceSkillsForSource, and their unified symlinks are pruned by the scanner.
func runRescan(only string) error {
	paths, cfg, err := config.Init()
	if err != nil {
		return err
	}
	db, err := database.Open(paths.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	sc := scanner.New(db, paths)
	sources, err := db.ListSources()
	if err != nil {
		return err
	}
	matched := false
	for _, src := range sources {
		if only != "" && src.ID != only {
			continue
		}
		matched = true
		skills, err := sc.ScanSource(src)
		if err != nil {
			return fmt.Errorf("rescan %s: %w", src.ID, err)
		}
		fmt.Printf("rescanned %s: %d skills\n", src.ID, len(skills))
	}
	if only != "" && !matched {
		return fmt.Errorf("source %q not found", only)
	}
	managed := activation.New(db, paths, cfg)
	removed, err := managed.CleanupDanglingManagedLinks()
	if err != nil {
		return fmt.Errorf("cleanup dangling activation links: %w", err)
	}
	if len(removed) > 0 {
		fmt.Printf("removed %d dangling activation link(s)\n", len(removed))
	}
	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
