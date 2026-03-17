package xdbsqlite

import "context"

// Health verifies the SQLite database is reachable and intact
// using PRAGMA quick_check.
func (s *Store) Health(ctx context.Context) error {
	var result string
	return s.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result)
}
