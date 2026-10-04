// Package migrations embeds the goose SQL migrations so they can be applied
// programmatically by the application and by integration tests, without
// requiring a goose binary.
package migrations

import "embed"

// FS holds all migration files.
//
//go:embed *.sql
var FS embed.FS
