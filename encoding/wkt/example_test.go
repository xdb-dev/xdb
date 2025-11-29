package wkt_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/xdb-dev/xdb/encoding/wkt"
)

func ExampleNewRegistry() {
	registry := wkt.NewRegistry()
	fmt.Println("Registry created:", registry.Count() == 0)
	// Output:
	// Registry created: true
}

func ExampleRegistry_Register() {
	type Money struct {
		Amount   int64
		Currency string
	}

	registry := wkt.NewRegistry()
	registry.Register(
		reflect.TypeOf(Money{}),
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

	fmt.Println("Type registered:", registry.IsRegistered(reflect.TypeOf(Money{})))
	// Output:
	// Type registered: true
}

func ExampleRegistry_Marshal() {
	type Money struct {
		Amount   int64
		Currency string
	}

	registry := wkt.NewRegistry()
	registry.Register(
		reflect.TypeOf(Money{}),
		func(v reflect.Value) (any, error) {
			m := v.Interface().(Money)
			return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
		},
		func(v any, target reflect.Value) error {
			return nil
		},
	)

	money := Money{Amount: 10000, Currency: "USD"}
	result, _ := registry.Marshal(reflect.ValueOf(money))
	fmt.Println(result)
	// Output:
	// 10000 USD
}

func ExampleRegistry_Unmarshal() {
	type Money struct {
		Amount   int64
		Currency string
	}

	registry := wkt.NewRegistry()
	registry.Register(
		reflect.TypeOf(Money{}),
		func(v reflect.Value) (any, error) {
			return nil, nil
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

	var money Money
	target := reflect.ValueOf(&money).Elem()
	_ = registry.Unmarshal("5000 EUR", target)
	fmt.Printf("%d %s\n", money.Amount, money.Currency)
	// Output:
	// 5000 EUR
}

func ExampleDefaultRegistry() {
	now := time.Date(2025, 11, 28, 12, 0, 0, 0, time.UTC)
	result, _ := wkt.DefaultRegistry.Marshal(reflect.ValueOf(now))
	fmt.Println(result)
	// Output:
	// 2025-11-28T12:00:00Z
}

func ExampleDefaultRegistry_duration() {
	duration := 5 * time.Second
	result, _ := wkt.DefaultRegistry.Marshal(reflect.ValueOf(duration))
	fmt.Println(result)
	// Output:
	// 5000000000
}

func ExampleRegistry_IsRegistered() {
	registry := wkt.NewRegistry()
	timeType := reflect.TypeOf(time.Time{})

	fmt.Println("Before:", registry.IsRegistered(timeType))

	registry.Register(
		timeType,
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)

	fmt.Println("After:", registry.IsRegistered(timeType))
	// Output:
	// Before: false
	// After: true
}

func ExampleRegistry_Count() {
	type Money struct {
		Amount   int64
		Currency string
	}

	registry := wkt.NewRegistry()
	fmt.Println("Initial count:", registry.Count())

	registry.Register(
		reflect.TypeOf(Money{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)
	fmt.Println("After registration:", registry.Count())

	registry.Register(
		reflect.TypeOf(time.Time{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)
	fmt.Println("After second registration:", registry.Count())
	// Output:
	// Initial count: 0
	// After registration: 1
	// After second registration: 2
}

func ExampleRegistry_Unregister() {
	type Money struct {
		Amount   int64
		Currency string
	}

	registry := wkt.NewRegistry()
	moneyType := reflect.TypeOf(Money{})

	registry.Register(
		moneyType,
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)

	fmt.Println("Registered:", registry.IsRegistered(moneyType))
	registry.Unregister(moneyType)
	fmt.Println("After unregister:", registry.IsRegistered(moneyType))
	// Output:
	// Registered: true
	// After unregister: false
}

func ExampleRegistry_Clear() {
	registry := wkt.NewRegistry()

	registry.Register(
		reflect.TypeOf(time.Time{}),
		func(v reflect.Value) (any, error) { return nil, nil },
		func(v any, target reflect.Value) error { return nil },
	)

	fmt.Println("Before clear:", registry.Count())
	registry.Clear()
	fmt.Println("After clear:", registry.Count())
	// Output:
	// Before clear: 1
	// After clear: 0
}

func ExampleRegistry_Enable() {
	registry := wkt.NewRegistry()

	fmt.Println("Initial:", registry.IsEnabled())
	registry.Disable()
	fmt.Println("Disabled:", registry.IsEnabled())
	registry.Enable()
	fmt.Println("Enabled:", registry.IsEnabled())
	// Output:
	// Initial: true
	// Disabled: false
	// Enabled: true
}
