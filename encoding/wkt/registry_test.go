package wkt_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/encoding/wkt"
)

// Custom type for testing
type Money struct {
	Amount   int64
	Currency string
}

func TestNewRegistry(t *testing.T) {
	r := wkt.NewRegistry()
	assert.NotNil(t, r)
	assert.Equal(t, 0, r.Count())
	assert.True(t, r.IsEnabled())
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name          string
		typ           reflect.Type
		marshal       wkt.MarshalFunc
		unmarshal     wkt.UnmarshalFunc
		shouldPanic   bool
		panicContains string
	}{
		{
			name: "ValidRegistration",
			typ:  reflect.TypeOf(Money{}),
			marshal: func(v reflect.Value) (any, error) {
				m := v.Interface().(Money)
				return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
			},
			unmarshal: func(v any, target reflect.Value) error {
				return nil
			},
			shouldPanic: false,
		},
		{
			name:          "NilType",
			typ:           nil,
			marshal:       func(v reflect.Value) (any, error) { return nil, nil },
			unmarshal:     func(v any, target reflect.Value) error { return nil },
			shouldPanic:   true,
			panicContains: "wkt: cannot register nil type",
		},
		{
			name:          "NilMarshal",
			typ:           reflect.TypeOf(Money{}),
			marshal:       nil,
			unmarshal:     func(v any, target reflect.Value) error { return nil },
			shouldPanic:   true,
			panicContains: "wkt: marshal function cannot be nil",
		},
		{
			name:          "NilUnmarshal",
			typ:           reflect.TypeOf(Money{}),
			marshal:       func(v reflect.Value) (any, error) { return nil, nil },
			unmarshal:     nil,
			shouldPanic:   true,
			panicContains: "wkt: unmarshal function cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := wkt.NewRegistry()

			if tt.shouldPanic {
				assert.PanicsWithValue(t, tt.panicContains, func() {
					r.Register(tt.typ, tt.marshal, tt.unmarshal)
				})
			} else {
				r.Register(tt.typ, tt.marshal, tt.unmarshal)
				assert.True(t, r.IsRegistered(tt.typ))
				assert.Equal(t, 1, r.Count())
			}
		})
	}
}

func TestRegisterOverwrite(t *testing.T) {
	r := wkt.NewRegistry()
	typ := reflect.TypeOf(Money{})

	marshal1 := func(v reflect.Value) (any, error) {
		return "version1", nil
	}
	unmarshal1 := func(v any, target reflect.Value) error {
		return nil
	}

	marshal2 := func(v reflect.Value) (any, error) {
		return "version2", nil
	}
	unmarshal2 := func(v any, target reflect.Value) error {
		return nil
	}

	r.Register(typ, marshal1, unmarshal1)
	assert.Equal(t, 1, r.Count())

	result1, _ := r.Marshal(reflect.ValueOf(Money{}))
	assert.Equal(t, "version1", result1)

	r.Register(typ, marshal2, unmarshal2)
	assert.Equal(t, 1, r.Count())

	result2, _ := r.Marshal(reflect.ValueOf(Money{}))
	assert.Equal(t, "version2", result2)
}

func TestUnregister(t *testing.T) {
	r := wkt.NewRegistry()
	typ := reflect.TypeOf(Money{})

	marshal := func(v reflect.Value) (any, error) {
		return "test", nil
	}
	unmarshal := func(v any, target reflect.Value) error {
		return nil
	}

	assert.False(t, r.Unregister(typ))

	r.Register(typ, marshal, unmarshal)
	assert.True(t, r.IsRegistered(typ))
	assert.Equal(t, 1, r.Count())

	assert.True(t, r.Unregister(typ))
	assert.False(t, r.IsRegistered(typ))
	assert.Equal(t, 0, r.Count())

	assert.False(t, r.Unregister(typ))
}

func TestIsRegistered(t *testing.T) {
	r := wkt.NewRegistry()
	moneyType := reflect.TypeOf(Money{})
	timeType := reflect.TypeOf(time.Time{})

	assert.False(t, r.IsRegistered(moneyType))
	assert.False(t, r.IsRegistered(timeType))

	r.Register(moneyType,
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)

	assert.True(t, r.IsRegistered(moneyType))
	assert.False(t, r.IsRegistered(timeType))
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*wkt.Registry)
		input       Money
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "SuccessfulMarshal",
			setup: func(r *wkt.Registry) {
				r.Register(
					reflect.TypeOf(Money{}),
					func(v reflect.Value) (any, error) {
						m := v.Interface().(Money)
						return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
					},
					func(v any, target reflect.Value) error { return nil },
				)
			},
			input:       Money{Amount: 10000, Currency: "USD"},
			expected:    "10000 USD",
			expectError: false,
		},
		{
			name:        "UnregisteredType",
			setup:       func(r *wkt.Registry) {},
			input:       Money{Amount: 10000, Currency: "USD"},
			expectError: true,
			errorMsg:    "not registered",
		},
		{
			name: "MarshalError",
			setup: func(r *wkt.Registry) {
				r.Register(
					reflect.TypeOf(Money{}),
					func(v reflect.Value) (any, error) {
						return nil, fmt.Errorf("marshal failed")
					},
					func(v any, target reflect.Value) error { return nil },
				)
			},
			input:       Money{Amount: 10000, Currency: "USD"},
			expectError: true,
			errorMsg:    "marshal failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := wkt.NewRegistry()
			tt.setup(r)

			result, err := r.Marshal(reflect.ValueOf(tt.input))

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*wkt.Registry)
		input       string
		expected    Money
		expectError bool
		errorMsg    string
	}{
		{
			name: "SuccessfulUnmarshal",
			setup: func(r *wkt.Registry) {
				r.Register(
					reflect.TypeOf(Money{}),
					func(v reflect.Value) (any, error) { return nil, nil },
					func(v any, target reflect.Value) error {
						str := v.(string)
						target.Set(reflect.ValueOf(Money{Amount: 10000, Currency: "USD"}))
						_ = str
						return nil
					},
				)
			},
			input:       "10000 USD",
			expected:    Money{Amount: 10000, Currency: "USD"},
			expectError: false,
		},
		{
			name:        "UnregisteredType",
			setup:       func(r *wkt.Registry) {},
			input:       "10000 USD",
			expectError: true,
			errorMsg:    "not registered",
		},
		{
			name: "UnmarshalError",
			setup: func(r *wkt.Registry) {
				r.Register(
					reflect.TypeOf(Money{}),
					func(v reflect.Value) (any, error) { return nil, nil },
					func(v any, target reflect.Value) error {
						return fmt.Errorf("unmarshal failed")
					},
				)
			},
			input:       "10000 USD",
			expectError: true,
			errorMsg:    "unmarshal failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := wkt.NewRegistry()
			tt.setup(r)

			var result Money
			target := reflect.ValueOf(&result).Elem()
			err := r.Unmarshal(tt.input, target)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEnableDisable(t *testing.T) {
	r := wkt.NewRegistry()

	assert.True(t, r.IsEnabled())

	r.Disable()
	assert.False(t, r.IsEnabled())

	r.Enable()
	assert.True(t, r.IsEnabled())

	r.Disable()
	assert.False(t, r.IsEnabled())
}

func TestCount(t *testing.T) {
	r := wkt.NewRegistry()
	assert.Equal(t, 0, r.Count())

	r.Register(
		reflect.TypeOf(Money{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)
	assert.Equal(t, 1, r.Count())

	r.Register(
		reflect.TypeOf(time.Time{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)
	assert.Equal(t, 2, r.Count())

	r.Unregister(reflect.TypeOf(Money{}))
	assert.Equal(t, 1, r.Count())

	r.Clear()
	assert.Equal(t, 0, r.Count())
}

func TestClear(t *testing.T) {
	r := wkt.NewRegistry()

	r.Register(
		reflect.TypeOf(Money{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)
	r.Register(
		reflect.TypeOf(time.Time{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)

	assert.Equal(t, 2, r.Count())

	r.Clear()
	assert.Equal(t, 0, r.Count())
	assert.False(t, r.IsRegistered(reflect.TypeOf(Money{})))
	assert.False(t, r.IsRegistered(reflect.TypeOf(time.Time{})))
}

func TestDefaultRegistry(t *testing.T) {
	assert.NotNil(t, wkt.DefaultRegistry)
	assert.True(t, wkt.DefaultRegistry.IsEnabled())
	assert.Greater(t, wkt.DefaultRegistry.Count(), 0, "DefaultRegistry should have built-in types")
}

func TestDefaultRegistryTimeTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "NonZeroTime",
			input:    time.Date(2025, 11, 28, 12, 0, 0, 0, time.UTC),
			expected: "2025-11-28T12:00:00Z",
		},
		{
			name:     "ZeroTime",
			input:    time.Time{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := wkt.DefaultRegistry.Marshal(reflect.ValueOf(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			var decoded time.Time
			target := reflect.ValueOf(&decoded).Elem()
			err = wkt.DefaultRegistry.Unmarshal(result, target)
			require.NoError(t, err)

			if tt.input.IsZero() {
				assert.True(t, decoded.IsZero())
			} else {
				assert.True(t, tt.input.Equal(decoded))
			}
		})
	}
}

func TestDefaultRegistryTimeDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected int64
	}{
		{
			name:     "PositiveDuration",
			input:    5 * time.Second,
			expected: int64(5 * time.Second),
		},
		{
			name:     "ZeroDuration",
			input:    0,
			expected: 0,
		},
		{
			name:     "NegativeDuration",
			input:    -10 * time.Minute,
			expected: int64(-10 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := wkt.DefaultRegistry.Marshal(reflect.ValueOf(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			var decoded time.Duration
			target := reflect.ValueOf(&decoded).Elem()
			err = wkt.DefaultRegistry.Unmarshal(result, target)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestDefaultRegistryDurationUnmarshalTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected time.Duration
	}{
		{
			name:     "Int64",
			input:    int64(5000000000),
			expected: 5 * time.Second,
		},
		{
			name:     "Int",
			input:    int(5000000000),
			expected: 5 * time.Second,
		},
		{
			name:     "Float64",
			input:    float64(5000000000),
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded time.Duration
			target := reflect.ValueOf(&decoded).Elem()
			err := wkt.DefaultRegistry.Unmarshal(tt.input, target)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, decoded)
		})
	}
}

func TestDefaultRegistryDurationUnmarshalError(t *testing.T) {
	var decoded time.Duration
	target := reflect.ValueOf(&decoded).Elem()
	err := wkt.DefaultRegistry.Unmarshal("invalid", target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected numeric value")
}

func TestDefaultRegistryTimeUnmarshalError(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		errorMsg string
	}{
		{
			name:     "WrongType",
			input:    12345,
			errorMsg: "expected string",
		},
		{
			name:     "InvalidFormat",
			input:    "not-a-time",
			errorMsg: "failed to parse time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded time.Time
			target := reflect.ValueOf(&decoded).Elem()
			err := wkt.DefaultRegistry.Unmarshal(tt.input, target)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := wkt.NewRegistry()
	typ := reflect.TypeOf(Money{})

	marshal := func(v reflect.Value) (any, error) {
		m := v.Interface().(Money)
		return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
	}
	unmarshal := func(v any, target reflect.Value) error {
		return nil
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			r.Register(typ, marshal, unmarshal)
			r.IsRegistered(typ)
			r.Count()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, r.IsRegistered(typ))
	assert.Equal(t, 1, r.Count())
}

func TestRoundTripCustomType(t *testing.T) {
	r := wkt.NewRegistry()
	typ := reflect.TypeOf(Money{})

	r.Register(
		typ,
		func(v reflect.Value) (any, error) {
			m := v.Interface().(Money)
			return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
		},
		func(v any, target reflect.Value) error {
			str := v.(string)
			var amount int64
			var currency string
			fmt.Sscanf(str, "%d %s", &amount, &currency)
			target.Set(reflect.ValueOf(Money{Amount: amount, Currency: currency}))
			return nil
		},
	)

	original := Money{Amount: 99999, Currency: "EUR"}

	marshaled, err := r.Marshal(reflect.ValueOf(original))
	require.NoError(t, err)
	assert.Equal(t, "99999 EUR", marshaled)

	var decoded Money
	target := reflect.ValueOf(&decoded).Elem()
	err = r.Unmarshal(marshaled, target)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}
