package xdbsqlite

import (
	"context"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

var (
	_ driver.RepoDriver = (*Store)(nil)
)

var (
	// ErrInvalidMode is returned when an invalid repo mode is encountered.
	ErrInvalidMode = errors.New("[xdbsqlite] invalid repo mode, allowed values: 'flexible', 'strict'")
	// ErrModeMismatch is returned when a repo has a different mode than the one being created.
	ErrModeMismatch = errors.New("[xdbsqlite] repo mode mismatch")
)

// MakeRepo creates or updates the table for the given schema.
func (s *Store) MakeRepo(ctx context.Context, repo *core.Repo) error {
	existing, err := s.GetRepo(ctx, repo.Name())
	if err != nil && !errors.Is(err, driver.ErrNotFound) {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if existing != nil && existing.Mode() != repo.Mode() {
		return errors.Wrap(ErrModeMismatch,
			"repo", repo.Name(),
			"existing", string(existing.Mode()),
			"new", string(repo.Mode()),
		)
	}

	r := &Migrator{tx: tx}

	mode := repo.Mode()

	switch mode {
	case core.ModeFlexible:
		err = r.CreateKeyValueTable(ctx, repo.Name())
		if err != nil {
			return err
		}
	case core.ModeStrict:
		err = r.CreateTable(ctx, repo.Name(), repo.Schema())
		if err != nil {
			return err
		}
	default:
		return errors.Wrap(ErrInvalidMode, "mode", string(mode))
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return s.meta.MakeRepo(ctx, repo)
}

func (s *Store) DeleteRepo(ctx context.Context, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := &Migrator{tx: tx}

	err = r.DropTable(ctx, name)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return s.meta.DeleteRepo(ctx, name)
}

func (s *Store) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	return s.meta.GetRepo(ctx, name)
}

func (s *Store) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	return s.meta.ListRepos(ctx)
}
