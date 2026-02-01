// Package config handles configuration path resolution for the SRS tool.
// It determines the appropriate location for the database and configuration files
// based on priority: user-specified paths, repository-local directories, or
// fallback to user home directory. It also provides utilities for resolving
// the cards directory path.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveConfigPath returns the config directory path based on priority:
// 1. User-specified flag (passed as argument)
// 2. Repo-local: .srs/ directory (if exists) or project root
// 3. Fallback: ~/.local/share/srs/
func ResolveConfigPath(userSpecified string) (string, error) {
	// Priority 1: User-specified path
	if userSpecified != "" {
		absPath, err := filepath.Abs(userSpecified)
		if err != nil {
			return "", fmt.Errorf("invalid config path: %w", err)
		}
		if err := ensureDir(absPath); err != nil {
			return "", err
		}
		return absPath, nil
	}

	// Priority 2: Repo-local (.srs/ directory or project root)
	// Check if we're in a git repo or have a .srs directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check for .srs directory
	localConfig := filepath.Join(wd, ".srs")
	if info, err := os.Stat(localConfig); err == nil && info.IsDir() {
		return localConfig, nil
	}

	// Check if we're in a git repo (look for .git directory)
	// If so, use project root
	gitRoot := findGitRoot(wd)
	if gitRoot != "" {
		// Use project root for config
		return gitRoot, nil
	}

	// Priority 3: Fallback to ~/.local/share/srs/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	fallbackPath := filepath.Join(homeDir, ".local", "share", "srs")
	if err := ensureDir(fallbackPath); err != nil {
		return "", err
	}

	return fallbackPath, nil
}

// DatabasePath returns the full path to the SQLite database file
func DatabasePath(configDir string) string {
	return filepath.Join(configDir, "srs.db")
}

// CardsPath returns the path to the cards directory.
// If userSpecified is provided, it uses that path (from flag or env var).
// Otherwise, defaults to "cards/" in the current working directory.
func CardsPath(userSpecified string) (string, error) {
	// Priority 1: User-specified path (from flag or env var)
	if userSpecified != "" {
		absPath, err := filepath.Abs(userSpecified)
		if err != nil {
			return "", fmt.Errorf("invalid cards path: %w", err)
		}
		if err := ensureDir(absPath); err != nil {
			return "", err
		}
		return absPath, nil
	}

	// Priority 2: Default to "cards/" in current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	cardsPath := filepath.Join(wd, "cards")
	if err := ensureDir(cardsPath); err != nil {
		return "", err
	}

	return cardsPath, nil
}

// ensureDir creates the directory if it doesn't exist
func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// findGitRoot walks up the directory tree to find .git directory
func findGitRoot(startPath string) string {
	dir := startPath
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}
	return ""
}
