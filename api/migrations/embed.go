// Package migrations holds the SQL migrations, embedded so the migrate
// command can run them without the files on disk.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
