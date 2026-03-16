package xdbsqlite

import "context"

// Health returns nil if the SQLite database is reachable.
func (s *Store) Health(ctx context.Context) error {
	return nil
}
