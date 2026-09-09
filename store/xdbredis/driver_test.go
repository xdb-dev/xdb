package xdbredis_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/storetest"
)

func redisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return "localhost:6379"
	}
	return addr
}

// testSeq is incremented to give each driver a unique prefix,
// preventing collisions between tests sharing one Redis.
var testSeq int

// newTestDriver creates a driver with a unique key prefix and registers
// cleanup of all keys under that prefix.
func newTestDriver(t testing.TB) *xdbredis.Driver {
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

	d := xdbredis.NewDriver(client, xdbredis.WithPrefix(prefix))

	// Verify connectivity.
	require.NoError(t, d.Health(context.Background()))

	return d
}

func TestDriverSuite(t *testing.T) {
	suite := storetest.NewDriverSuite(func() store.Driver {
		return newTestDriver(t)
	})
	suite.Run(t)
}

func TestQuerySuite(t *testing.T) {
	storetest.NewQuerySuite(func() store.Driver {
		return newTestDriver(t)
	}).Run(t)
}
