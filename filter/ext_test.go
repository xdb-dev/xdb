package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func eventDef() *schema.Def {
	return &schema.Def{
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"start": {Type: core.TypeTime},
		},
	}
}

func eventRecord(title string, start time.Time) *core.Record {
	record := core.NewRecord("cal", "events", "1")
	record.Set("title", title)
	record.Set("start", start)

	return record
}

func TestMatch_Extensions(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	soon := time.Now().Add(10 * time.Minute)

	tests := []struct {
		name   string
		expr   string
		record *core.Record
		want   bool
		typed  bool // the case needs a schema that types start as a time
	}{
		{
			name:   "lowerAscii",
			expr:   `title.lowerAscii() == "standup"`,
			record: eventRecord("StandUp", start),
			want:   true,
		},
		{
			name:   "upperAscii contains",
			expr:   `title.upperAscii().contains("STAND")`,
			record: eventRecord("standup", start),
			want:   true,
		},
		{
			name:   "now is after a past time",
			expr:   `start < now`,
			record: eventRecord("standup", past),
			want:   true,
		},
		{
			name:   "now plus a duration",
			expr:   `start > now && start <= now + duration("15m")`,
			record: eventRecord("standup", soon),
			want:   true,
		},
		{
			name:   "now plus a shorter duration",
			expr:   `start <= now + duration("5m")`,
			record: eventRecord("standup", soon),
			want:   false,
		},
		{
			name:   "time at or after an RFC 3339 string",
			expr:   `start >= "2026-09-29T00:00:00Z"`,
			record: eventRecord("standup", start),
			want:   true,
			typed:  true,
		},
		{
			name:   "time before an RFC 3339 string",
			expr:   `start < "2026-09-29T00:00:00Z"`,
			record: eventRecord("standup", start),
			want:   false,
			typed:  true,
		},
		{
			name:   "RFC 3339 string on the left",
			expr:   `"2026-09-30T00:00:00Z" > start`,
			record: eventRecord("standup", start),
			want:   true,
			typed:  true,
		},
		{
			name:   "time equals an RFC 3339 string",
			expr:   `start == "2026-09-29T10:00:00Z"`,
			record: eventRecord("standup", start),
			want:   true,
			typed:  true,
		},
		{
			name:   "time not equal to an RFC 3339 string",
			expr:   `start != "2026-09-29T10:00:00Z"`,
			record: eventRecord("standup", start),
			want:   false,
			typed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defs := []*schema.Def{eventDef(), nil}
			if tt.typed {
				defs = defs[:1]
			}

			for _, def := range defs {
				f, err := Compile(tt.expr, def)
				require.NoError(t, err)

				got, err := f.Match(tt.record)
				require.NoError(t, err)
				assert.Equal(t, tt.want, got, "def=%v", def != nil)
			}
		})
	}
}

func TestCompile_RejectsAMalformedTimeString(t *testing.T) {
	t.Parallel()

	_, err := Compile(`start >= "29-09-2026"`, eventDef())
	require.ErrorIs(t, err, core.ErrInvalidFilter)
	assert.Contains(t, err.Error(), "RFC 3339")
}

func TestCompile_StringComparisonIsStillAStringComparison(t *testing.T) {
	t.Parallel()

	f, err := Compile(`title >= "m"`, eventDef())
	require.NoError(t, err)

	got, err := f.Match(eventRecord("standup", time.Now()))
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCompile_AFieldNamedNowWins(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"now": {Type: core.TypeString},
		},
	}

	f, err := Compile(`now == "later"`, def)
	require.NoError(t, err)

	record := core.NewRecord("cal", "events", "1")
	record.Set("now", "later")

	got, err := f.Match(record)
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCompile_NowIsTheClockWithoutADeclaredField(t *testing.T) {
	t.Parallel()

	f, err := Compile(`now > timestamp("2026-01-01T00:00:00Z")`, nil)
	require.NoError(t, err)

	record := core.NewRecord("cal", "events", "1")
	record.Set("now", "a string attribute")

	got, err := f.Match(record)
	require.NoError(t, err)
	assert.True(t, got)
}

func TestCompile_AStringAgainstAnUntypedFieldStaysAString(t *testing.T) {
	t.Parallel()

	f, err := Compile(`start >= "2026-09-29T00:00:00Z"`, nil)
	require.NoError(t, err)

	got, err := f.Match(eventRecord("standup", time.Now()))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestFilter_Clock(t *testing.T) {
	withNow := eventDef()
	withNow.Fields[NowVar] = schema.Field{Type: core.TypeString}

	tests := []struct {
		name string
		expr string
		def  *schema.Def
		want bool
	}{
		{name: "namespace scope", expr: `title == "a"`, def: nil, want: true},
		{name: "schema without now", expr: `title == "a"`, def: eventDef(), want: true},
		{name: "schema declares now", expr: `now == "later"`, def: withNow, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Compile(tt.expr, tt.def)
			require.NoError(t, err)
			assert.Equal(t, tt.want, f.Clock())
		})
	}
}
