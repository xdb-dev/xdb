// Package xdbfs provides a filesystem-based driver implementation for XDB.
//
// The filesystem driver stores XDB data as JSON files on the local filesystem.
// It implements the SchemaDriver, TupleDriver, and RecordDriver interfaces.
//
// Directory Structure:
//
//	{root}/
//	├── {namespace}/
//	│   ├── {schema_name}/
//	│   │   ├── .schema.json    # Schema definition file
//	│   │   ├── {id1}.json      # Record file for ID "id1"
//	│   │   ├── {id2}.json      # Record file for ID "id2"
//	│   │   └── ...
//	│   └── ...
//	└── ...
//
// Each record is stored as a separate JSON file where:
//   - The filename is {id}.json (with "/" in IDs replaced by "_")
//   - The file contains a JSON object mapping attribute names to encoded values
//   - Values are encoded using the codec/json package
//
// Schemas are stored as .schema.json files in the schema directory.
//
// Example Usage:
//
//	driver, err := xdbfs.New("/path/to/data")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Put a record
//	record := core.NewRecord("default", "users", "user1")
//	record.Set("name", "Alice")
//	record.Set("age", int64(30))
//	err = driver.PutRecords(ctx, []*core.Record{record})
//
//	// Get a record
//	uri, _ := core.ParseURI("xdb://default/users/user1")
//	records, _, err := driver.GetRecords(ctx, []*core.URI{uri})
package xdbfs
