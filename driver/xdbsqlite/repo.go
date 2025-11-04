package xdbsqlite

import (
	"context"

	"github.com/gojekfarm/xtools/errors"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

var (
	ErrUnsupportedType = errors.New("xdb/driver/xdbsqlite: unsupported type")
	ErrFieldDeleted    = errors.New("xdb/driver/xdbsqlite: deleting fields is not supported")
	ErrFieldModified   = errors.New("xdb/driver/xdbsqlite: modifying field type or constraints is not supported")
)

func (s *Store) MakeRepo(ctx context.Context, repo *core.Repo) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	migrator := NewMigrator(tx)

	err = migrator.CreateTable(ctx, repo.Schema())
	if err != nil {
		return err
	}

	err = s.meta.CreateRepo(ctx, repo)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) UpdateRepo(ctx context.Context, repo *core.Repo) error {
	existing, err := s.meta.GetRepo(ctx, repo.Name())
	if err != nil {
		return err
	}

	if existing == nil {
		return driver.ErrRepoNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	migrator := NewMigrator(tx)

	err = migrator.AlterTable(ctx, existing.Schema(), repo.Schema())
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	return s.meta.GetRepo(ctx, name)
}

func (s *Store) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	return s.meta.ListRepos(ctx)
}

func (s *Store) DeleteRepo(ctx context.Context, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	migrator := NewMigrator(tx)

	err = migrator.DropTable(ctx, name)
	if err != nil {
		return err
	}

	err = s.meta.DeleteRepo(ctx, name)
	if err != nil {
		return err
	}

	return tx.Commit()
}
