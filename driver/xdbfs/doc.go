// Package xdbfs provides a filesystem-based driver implementation for XDB.
//
// Directory structure:
//
//	{repo_name}/
//	├── .schema.json # Schema file associated with the repository
//	├── {id1}.json   # Record file for record with ID "id1"
//	├── {id1}/
//	│   ├── {nested-id2}.json # Record file for record with ID "id1/nested-id2"
//	│   └── ...
//	└── ...
package xdbfs
