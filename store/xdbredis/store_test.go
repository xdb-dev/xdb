package xdbredis_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store/xdbredis"
)

func redisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return "localhost:6379"
	}
	return addr
}

// testSeq is incremented to give each test a unique prefix, preventing collisions.
var testSeq int

func newTestStore(t *testing.T) *xdbredis.Store {
	t.Helper()

	testSeq++
	prefix := fmt.Sprintf("xdbtest:%s:%d", t.Name(), testSeq)

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr(),
	})

	t.Cleanup(func() {
		ctx := context.Background()
		// Clean up all keys with this prefix.
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, prefix+":*", 100).Result()
			if err != nil {
				break
			}
			if len(keys) > 0 {
				client.Del(ctx, keys...)
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		client.Close()
	})

	s := xdbredis.New(client, xdbredis.WithPrefix(prefix))

	// Verify connectivity.
	require.NoError(t, s.Health(context.Background()))

	return s
}
