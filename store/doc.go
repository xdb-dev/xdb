// Package store defines the interfaces for XDB storage backends.
//
// The store package is the boundary between the service layer and
// storage implementations. The service layer is the only consumer
// of these interfaces — RPC handlers never touch the store directly.
//
// Store implementations must be safe for concurrent use.
package store
