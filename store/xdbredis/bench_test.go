package xdbredis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/tests"
)

var benchSeq int

func BenchmarkStore(b *testing.B) {
	tests.NewBenchmarkSuite(func() store.Store {
		benchSeq++
		prefix := fmt.Sprintf("xdbbench:%d", benchSeq)

		client := redis.NewClient(&redis.Options{
			Addr: redisAddr(),
		})

		b.Cleanup(func() {
			ctx := context.Background()
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
		if err := s.Health(context.Background()); err != nil {
			b.Skipf("redis not available: %v", err)
		}

		return s
	}).Run(b)
}
