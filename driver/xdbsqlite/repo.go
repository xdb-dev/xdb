package xdbsqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/xdb-dev/xdb/core"
)

var (
	ErrRepoAlreadyExists = errors.New("[xdb/driver/xdbsqlite] repo already exists")
	ErrRepoNotFound      = errors.New("[xdb/driver/xdbsqlite] repo not found")
)

type Manager struct {
	dir string

	mu   sync.Mutex
	pool map[string]*sql.DB
}

func NewRepoManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &Manager{dir: dir}, nil
}

func (m *Manager) GetRepo(ctx context.Context, uri *core.URI) (*core.Repo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.pool[uri.Repo()]; !ok {
		return nil, ErrRepoNotFound
	}

	return core.NewRepo(uri.Repo())
}

func (m *Manager) CreateRepo(ctx context.Context, repo *core.Repo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.pool[repo.Name()]; ok {
		return ErrRepoAlreadyExists
	}

	db, err := sql.Open("sqlite3", filepath.Join(m.dir, repo.Name()+".db"))
	if err != nil {
		return err
	}

	m.pool[repo.Name()] = db

	return nil
}

func (m *Manager) GetCollection(ctx context.Context, uri *core.URI) (*core.Collection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, ok := m.pool[uri.Repo()]
	if !ok {
		return nil, ErrRepoNotFound
	}

	_ = db // TODO: implement

	return &core.Collection{
		Repo:   uri.Repo(),
		ID:     uri.Collection(),
		Schema: &core.Schema{},
	}, nil
}

func (m *Manager) CreateCollection(ctx context.Context, collection *core.Collection) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, ok := m.pool[collection.Repo]
	if !ok {
		return ErrRepoNotFound
	}

	_ = db // TODO: implement

	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	errs := []error{}
	for name, db := range m.pool {
		if err := db.Close(); err != nil {
			errs = append(errs, err)
		}

		delete(m.pool, name)
	}

	return errors.Join(errs...)
}
