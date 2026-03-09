// Package types provides encoding and decoding between XDB's core type system
// and database-specific representations using [driver.Valuer] and [sql.Scanner].
//
// Each database backend creates a [Codec] and registers [Mapping] entries
// for the [core.Type] values it supports. The codec converts [core.Value]
// to [driver.Value] on writes and scans database values back to [core.Value]
// on reads.
//
// # Usage
//
// A backend creates a codec during initialization:
//
//	c := types.New("sqlite")
//	c.Register(types.Passthrough(core.TypeString, "TEXT"))
//	c.Register(types.Mapping{
//	    Type:     core.TypeBool,
//	    TypeName: "INTEGER",
//	    Value:    encodeBool,
//	    Scan:     decodeBool,
//	})
//
// Use [Codec.Value] and [Codec.Scan] directly:
//
//	dbVal, err := c.Value(value)
//	value, err := c.Scan(core.TypeString, src)
//
// Or use [Column] with database/sql for idiomatic row scanning:
//
//	col := c.Column(core.TypeString, stringVal)
//	db.Exec("INSERT INTO t (col) VALUES (?)", col)   // driver.Valuer
//
//	col := c.Column(core.TypeString, nil)
//	row.Scan(col)                                      // sql.Scanner
//	val := col.Val()
//
// # Custom Types
//
// Since [core.TID] is a string, custom type identifiers can be registered
// alongside built-in types:
//
//	const TIDMoney core.TID = "MONEY"
//	c.Register(types.Mapping{Type: core.NewType(TIDMoney), ...})
package types
