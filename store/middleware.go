package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// newLoggingDriver wraps next so driver writes are logged at debug
// level and failures at error level. Reads are inherited unchanged
// from the embedded [Driver].
func newLoggingDriver(logger *slog.Logger, next Driver) Driver {
	return &loggingDriver{Driver: next, logger: logger}
}

// loggingDriver logs write operations; reads pass through the embedded
// Driver untouched.
type loggingDriver struct {
	Driver
	logger *slog.Logger
}

func (l *loggingDriver) Apply(ctx context.Context, m Mutation) error {
	start := time.Now()
	err := l.Driver.Apply(ctx, m)
	l.log(ctx, "store.apply", err,
		slog.String("path", m.Path.String()),
		slog.String("op", m.Op.String()),
		slog.Duration("elapsed", time.Since(start)),
	)
	return err
}

func (l *loggingDriver) CreateSchema(ctx context.Context, def *schema.Def) error {
	err := l.Driver.CreateSchema(ctx, def)
	l.log(ctx, "store.create_schema", err, slog.String("uri", def.URI.String()))
	return err
}

func (l *loggingDriver) PutSchema(ctx context.Context, def *schema.Def) error {
	err := l.Driver.PutSchema(ctx, def)
	l.log(ctx, "store.put_schema", err, slog.String("uri", def.URI.String()))
	return err
}

func (l *loggingDriver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	err := l.Driver.DeleteSchema(ctx, uri)
	l.log(ctx, "store.delete_schema", err, slog.String("uri", uri.String()))
	return err
}

func (l *loggingDriver) DropRecords(ctx context.Context, uri *core.URI) error {
	err := l.Driver.DropRecords(ctx, uri)
	l.log(ctx, "store.drop_records", err, slog.String("uri", uri.String()))
	return err
}

func (l *loggingDriver) log(
	ctx context.Context,
	op string,
	err error,
	attrs ...slog.Attr,
) {
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		l.logger.LogAttrs(ctx, slog.LevelError, op, attrs...)
		return
	}
	l.logger.LogAttrs(ctx, slog.LevelDebug, op, attrs...)
}
