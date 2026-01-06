// Package xdbfs provides a filesystem-based store implementation for XDB.
//
// The filesystem store stores XDB data as JSON files on the local filesystem.
// It implements the SchemaStore, TupleStore, and RecordStore interfaces.
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
// File Permissions:
//
// By default, the filesystem store uses restrictive permissions for security:
//   - Directories: 0o750 (rwxr-x---)
//   - Files: 0o600 (rw-------)
//
// These can be customized using functional options:
//
//	// Use default restrictive permissions
//	store, err := xdbfs.New("/path/to/data")
//
//	// Use shared access permissions (0o755/0o644)
//	store, err := xdbfs.New("/path/to/data", xdbfs.WithSharedAccess())
//
//	// Custom permissions
//	store, err := xdbfs.New("/path/to/data", xdbfs.WithPermissions(0o770, 0o660))
//
// Example Usage:
//
//	store, err := xdbfs.New("/path/to/data")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Put a record
//	record := core.NewRecord("default", "users", "user1")
//	record.Set("name", "Alice")
//	record.Set("age", int64(30))
//	err = store.PutRecords(ctx, []*core.Record{record})
//
//	// Get a record
//	uri, _ := core.ParseURI("xdb://default/users/user1")
//	records, _, err := store.GetRecords(ctx, []*core.URI{uri})
package xdbfs
