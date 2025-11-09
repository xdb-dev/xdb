package xdbfs

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
)

// FS is a filesystem driver for XDB.
type FS struct {
	root string
}

// New creates a new FS driver.
func New(root string) (*FS, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	return &FS{root: root}, nil
}

// GetRepo gets the repo from the filesystem.
func (fs *FS) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	path := fs.repoPath(name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, driver.ErrNotFound
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	schema, err := schema.LoadFromJSON(data)
	if err != nil {
		return nil, err
	}

	repo, err := core.NewRepo(name)
	if err != nil {
		return nil, err
	}

	repo = repo.WithSchema(schema)

	return repo, nil
}

// ListRepos lists the repos from the filesystem.
func (fs *FS) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	repos := make([]*core.Repo, 0)

	err := filepath.Walk(fs.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".schema.json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		schema, err := schema.LoadFromJSON(data)
		if err != nil {
			return err
		}

		repo, err := core.NewRepo(schema.Name)
		if err != nil {
			return err
		}

		repos = append(repos, repo.WithSchema(schema))
		return nil
	})

	return repos, err
}

// MakeRepo creates a new repo in the filesystem.
func (fs *FS) MakeRepo(ctx context.Context, repo *core.Repo) error {
	path := fs.repoPath(repo.Name())

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := schema.WriteToJSON(repo.Schema())
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	return nil
}

// DeleteRepo deletes a repo from the filesystem.
func (fs *FS) DeleteRepo(ctx context.Context, name string) error {
	path := fs.repoPath(name)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return os.RemoveAll(path)
}

// repoPath returns the file path for a repository.
func (fs *FS) repoPath(name string) string {
	return filepath.Join(fs.root, name, ".schema.json")
}
