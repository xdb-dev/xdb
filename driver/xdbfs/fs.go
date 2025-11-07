package xdbfs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

// FS is a filesystem driver for XDB.
// It is used to store and retrieve repositories from a filesystem.
// Each repository is stored as a separate JSON file.
type FS struct {
	root string
}

// New creates a new FS driver.
func New(root string) *FS {
	if err := os.MkdirAll(root, 0755); err != nil {
		panic(fmt.Errorf("failed to create root directory: %w", err))
	}

	return &FS{root: root}
}

// GetRepo gets the repo from the filesystem.
func (fs *FS) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	path := fs.repoPath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, driver.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read repo file: %w", err)
	}

	var rf repoFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo: %w", err)
	}

	repo, err := core.NewRepo(rf.Name)
	if err != nil {
		return nil, err
	}

	repo = repo.WithSchema(rf.Schema)

	return repo, nil
}

// ListRepos lists the repos from the filesystem.
func (fs *FS) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	entries, err := os.ReadDir(fs.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []*core.Repo{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	repos := make([]*core.Repo, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		repoName := strings.TrimSuffix(name, ".json")
		repo, err := fs.GetRepo(ctx, repoName)
		if err != nil {
			return nil, fmt.Errorf("failed to load repo %s: %w", repoName, err)
		}

		repos = append(repos, repo)
	}

	return repos, nil
}

// CreateRepo creates a new repo in the filesystem.
func (fs *FS) CreateRepo(ctx context.Context, repo *core.Repo) error {
	// Ensure the root directory exists
	if err := os.MkdirAll(fs.root, 0755); err != nil {
		return fmt.Errorf("failed to create root directory: %w", err)
	}

	rf := repoFile{
		Name:   repo.Name(),
		Schema: repo.Schema(),
	}

	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repo: %w", err)
	}

	path := fs.repoPath(repo.Name())
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write repo file: %w", err)
	}

	return nil
}

// UpdateRepo updates a repo in the filesystem.
func (fs *FS) UpdateRepo(ctx context.Context, repo *core.Repo) error {
	path := fs.repoPath(repo.Name())

	data, err := json.MarshalIndent(repoFile{
		Name:   repo.Name(),
		Schema: repo.Schema(),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repo: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write repo file: %w", err)
	}

	return nil
}

// DeleteRepo deletes a repo from the filesystem.
func (fs *FS) DeleteRepo(ctx context.Context, name string) error {
	path := fs.repoPath(name)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete repo file: %w", err)
	}

	return nil
}

// repoPath returns the file path for a repository.
func (fs *FS) repoPath(name string) string {
	return filepath.Join(fs.root, name+".json")
}

// repoFile represents the JSON structure for a repository file.
type repoFile struct {
	Name   string       `json:"name"`
	Schema *core.Schema `json:"schema,omitempty"`
}
