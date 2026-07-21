package xdbredis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

const defaultPrefix = "xdb"

// defSuffix is the final key segment marking schema definition keys:
// {prefix}:{ns}:{schema}:_schema. A record whose ID is exactly
// "_schema" would collide with its schema's definition key; this
// collision is inherited from the original key layout.
const defSuffix = "_schema"

// Driver is a Redis-backed implementation of [store.Driver].
//
// It deliberately does not implement [store.TxDriver] — Redis has no
// native rollback-capable transactions — so the facade falls back to
// sequential writes.
type Driver struct {
	client redis.UniversalClient
	prefix string
}

// Option configures a [Driver].
type Option func(*Driver)

// WithPrefix sets the key prefix. Default: "xdb".
func WithPrefix(prefix string) Option {
	return func(d *Driver) {
		d.prefix = prefix
	}
}

// NewDriver creates a new Redis driver over the given client.
func NewDriver(client redis.UniversalClient, opts ...Option) *Driver {
	d := &Driver{
		client: client,
		prefix: defaultPrefix,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Close closes the underlying Redis client.
func (d *Driver) Close() error {
	return d.client.Close()
}

// Health checks Redis connectivity by sending a PING command.
func (d *Driver) Health(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

// --- Key helpers ---

// recordKey returns the Redis key for a record: {prefix}:{ns}:{schema}:{id}.
func (d *Driver) recordKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		d.prefix,
		uri.NS(),
		uri.Schema(),
		uri.ID(),
	)
}

// defKey returns the Redis key for a schema definition:
// {prefix}:{ns}:{schema}:_schema.
func (d *Driver) defKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		d.prefix,
		uri.NS(),
		uri.Schema(),
		defSuffix,
	)
}

// parseRecordKey is the inverse of recordKey: it extracts the record
// path (ns/schema/id) from a Redis key. It reports false for keys
// outside the prefix and for schema definition keys. URI components
// never contain ":" (only the ID may contain "/"), so splitting on
// ":" is unambiguous.
func (d *Driver) parseRecordKey(key string) (string, bool) {
	rest, found := strings.CutPrefix(key, d.prefix+":")
	if !found {
		return "", false
	}

	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 {
		return "", false
	}
	if parts[2] == defSuffix {
		return "", false
	}

	return parts[0] + "/" + parts[1] + "/" + parts[2], true
}

// globEscaper escapes Redis MATCH glob metacharacters so key segments
// (notably user-supplied prefixes) match literally.
var globEscaper = strings.NewReplacer(
	`\`, `\\`,
	`*`, `\*`,
	`?`, `\?`,
	`[`, `\[`,
	`]`, `\]`,
)

// scanKeys collects all keys matching pattern, sorted and deduplicated
// (SCAN guarantees at-least-once delivery, not exactly-once).
func (d *Driver) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := d.client.Scan(ctx, cursor, pattern, 256).Result()
		if err != nil {
			return nil, fmt.Errorf("xdbredis: scan %q: %w", pattern, err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	slices.Sort(keys)
	return slices.Compact(keys), nil
}

// --- Tuple reads ---

// GetTuples returns the tuples at the given attr-level URIs using a
// pipeline of HGET point reads. Absent attrs are omitted; results are
// in request order.
func (d *Driver) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	if len(uris) == 0 {
		return nil, nil
	}

	pipe := d.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(uris))
	for i, uri := range uris {
		cmds[i] = pipe.HGet(ctx, d.recordKey(uri), uri.Attr())
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("xdbredis: get tuples: %w", err)
	}

	got := make([]*core.Tuple, 0, len(uris))
	for i, cmd := range cmds {
		raw, err := cmd.Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("xdbredis: hget %s: %w", uris[i], err)
		}

		path := recordPath(uris[i])
		tuple, err := decodeTuple(path, uris[i].Attr(), raw)
		if err != nil {
			return nil, err
		}
		got = append(got, tuple)
	}
	return got, nil
}

// ScanTuples yields every tuple under scope: a namespace, a schema, or
// a record path. Record keys are sorted for determinism and each
// record's hash is read whole, so tuples of one record are contiguous.
func (d *Driver) ScanTuples(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return func(yield func(*core.Tuple, error) bool) {
		keys, err := d.recordKeys(ctx, scope)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, key := range keys {
			path, ok := d.parseRecordKey(key)
			if !ok {
				continue
			}

			fields, err := d.client.HGetAll(ctx, key).Result()
			if err != nil {
				yield(nil, fmt.Errorf("xdbredis: hgetall %s: %w", key, err))
				return
			}

			if !yieldRecord(yield, path, fields) {
				return
			}
		}
	}
}

// recordKeys returns the sorted record keys under scope. A record
// scope needs no SCAN — its single key is addressed directly.
func (d *Driver) recordKeys(ctx context.Context, scope *core.URI) ([]string, error) {
	if scope.ID() != "" {
		return []string{d.recordKey(scope)}, nil
	}

	literal := d.prefix + ":" + scope.NS()
	if scope.Schema() != "" {
		literal += ":" + scope.Schema()
	}
	return d.scanKeys(ctx, globEscaper.Replace(literal)+":*")
}

// yieldRecord decodes and yields one record's hash fields in sorted
// attr order. It reports whether iteration should continue.
func yieldRecord(
	yield func(*core.Tuple, error) bool,
	path string,
	fields map[string]string,
) bool {
	for _, attr := range slices.Sorted(maps.Keys(fields)) {
		tuple, err := decodeTuple(path, attr, fields[attr])
		if err != nil {
			yield(nil, err)
			return false
		}
		if !yield(tuple, nil) {
			return false
		}
	}
	return true
}

// recordPath returns the record path string (ns/schema/id) of a URI,
// ignoring any attr component.
func recordPath(uri *core.URI) string {
	return uri.RecordPath()
}

// decodeTuple decodes one hash field into a tuple at path.
func decodeTuple(path, attr, raw string) (*core.Tuple, error) {
	val, err := decodeValue(raw)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode field %s: %w", attr, err)
	}
	return core.NewTuple(path, attr, val), nil
}

// --- Tuple writes ---

// createScript atomically creates a record hash in a single round
// trip: it fails if the key already exists, otherwise writes all
// attr/value pairs. Pairs are written in a Lua loop instead of
// unpack() to avoid Lua's stack limit on large records.
var createScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
	return 0
end
for i = 1, #ARGV, 2 do
	redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1])
end
return 1
`)

// replaceScript atomically replaces a record hash unconditionally:
// delete then write, so attrs absent from the mutation are dropped.
var replaceScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
for i = 1, #ARGV, 2 do
	redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1])
end
return 1
`)

// Apply executes one mutation atomically: it is either a single Redis
// command (patch, delete) or one Lua script (create, put), so a
// failing mutation has no partial effect.
func (d *Driver) Apply(ctx context.Context, m store.Mutation) error {
	return d.applyMutation(ctx, m)
}

// applyMutation applies one mutation. A record with zero tuples does
// not exist: replace-style ops with no tuples leave no key behind, and
// Redis removes a hash automatically when HDEL deletes its last field.
func (d *Driver) applyMutation(ctx context.Context, m store.Mutation) error {
	key := d.recordKey(m.Path)

	switch m.Op {
	case store.OpPatch:
		args, err := encodeTuples(m.Tuples)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return nil
		}
		if err := d.client.HSet(ctx, key, args...).Err(); err != nil {
			return fmt.Errorf("xdbredis: merge: %w", err)
		}
		return nil

	case store.OpCreate:
		return d.runReplaceScript(ctx, createScript, key, m.Tuples, core.ErrAlreadyExists)

	case store.OpPut:
		return d.runReplaceScript(ctx, replaceScript, key, m.Tuples, nil)

	case store.OpDelete:
		if len(m.Attrs) == 0 {
			if err := d.client.Del(ctx, key).Err(); err != nil {
				return fmt.Errorf("xdbredis: delete: %w", err)
			}
			return nil
		}
		if err := d.client.HDel(ctx, key, m.Attrs...).Err(); err != nil {
			return fmt.Errorf("xdbredis: delete attrs: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("xdbredis: unknown op %s", m.Op)
	}
}

// runReplaceScript runs one of the replace-style Lua scripts. The
// scripts return 0 when their existence gate fails, which maps to
// gateErr (nil for the unconditional replace).
func (d *Driver) runReplaceScript(
	ctx context.Context,
	script *redis.Script,
	key string,
	tuples []*core.Tuple,
	gateErr error,
) error {
	args, err := encodeTuples(tuples)
	if err != nil {
		return err
	}

	ok, err := script.Run(ctx, d.client, []string{key}, args...).Int()
	if err != nil {
		return fmt.Errorf("xdbredis: apply: %w", err)
	}
	if ok == 0 {
		return gateErr
	}
	return nil
}

// encodeTuples flattens tuples into HSET-style attr/value argument
// pairs using the type-prefixed string codec.
func encodeTuples(tuples []*core.Tuple) ([]any, error) {
	args := make([]any, 0, len(tuples)*2)
	for _, t := range tuples {
		encoded, err := encodeValue(t.Value())
		if err != nil {
			return nil, fmt.Errorf("xdbredis: encode field %s: %w", t.Attr(), err)
		}
		args = append(args, t.Attr(), encoded)
	}
	return args, nil
}

// --- Definition reads ---

// GetSchema retrieves a definition by URI (ns + schema).
// Returns [core.ErrNotFound] if absent.
func (d *Driver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	data, err := d.client.Get(ctx, d.defKey(uri)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("xdbredis: get def: %w", err)
	}
	return decodeDef(data)
}

// ScanSchemas yields definitions under scope: nil for all, or a
// namespace URI. Defs are yielded in sorted key order.
func (d *Driver) ScanSchemas(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return func(yield func(*schema.Def, error) bool) {
		keys, err := d.scanKeys(ctx, d.defPattern(scope))
		if err != nil {
			yield(nil, err)
			return
		}

		for _, key := range keys {
			data, err := d.client.Get(ctx, key).Bytes()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				yield(nil, fmt.Errorf("xdbredis: get def %s: %w", key, err))
				return
			}

			def, err := decodeDef(data)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(def, nil) {
				return
			}
		}
	}
}

// defPattern returns the MATCH pattern for definition keys in scope.
func (d *Driver) defPattern(scope *core.URI) string {
	literal := d.prefix
	if scope != nil {
		literal += ":" + scope.NS()
	}
	return globEscaper.Replace(literal) + ":*:" + defSuffix
}

// decodeDef unmarshals a stored definition.
func decodeDef(data []byte) (*schema.Def, error) {
	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("xdbredis: unmarshal def: %w", err)
	}
	return &def, nil
}

// --- Definition writes ---

// CreateSchema stores a new definition verbatim. Returns
// [core.ErrAlreadyExists] if one exists — the existence gate is a
// single atomic SET NX.
func (d *Driver) CreateSchema(ctx context.Context, def *schema.Def) error {
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal def: %w", err)
	}

	ok, err := d.client.SetNX(ctx, d.defKey(def.URI), data, 0).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: create def: %w", err)
	}
	if !ok {
		return core.ErrAlreadyExists
	}
	return nil
}

// PutSchema stores a definition verbatim and unconditionally (upsert).
func (d *Driver) PutSchema(ctx context.Context, def *schema.Def) error {
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal def: %w", err)
	}

	if err := d.client.Set(ctx, d.defKey(def.URI), data, 0).Err(); err != nil {
		return fmt.Errorf("xdbredis: put def: %w", err)
	}
	return nil
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if absent.
func (d *Driver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	n, err := d.client.Del(ctx, d.defKey(uri)).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: delete def: %w", err)
	}
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}

// DropRecords deletes all record keys belonging to a schema,
// keeping the definition itself. No-op if no records exist.
func (d *Driver) DropRecords(ctx context.Context, uri *core.URI) error {
	literal := d.prefix + ":" + uri.NS() + ":" + uri.Schema()
	keys, err := d.scanKeys(ctx, globEscaper.Replace(literal)+":*")
	if err != nil {
		return err
	}

	defKey := d.defKey(uri)
	records := slices.DeleteFunc(keys, func(key string) bool {
		return key == defKey
	})
	if len(records) == 0 {
		return nil
	}

	if err := d.client.Del(ctx, records...).Err(); err != nil {
		return fmt.Errorf("xdbredis: delete schema records: %w", err)
	}
	return nil
}
