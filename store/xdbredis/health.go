package xdbredis

import "context"

// Health checks Redis connectivity by sending a PING command.
func (s *Store) Health(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
