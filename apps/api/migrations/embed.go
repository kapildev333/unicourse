// Package migrations embeds the SQL migration files.
package migrations

import "embed"

// FS holds *.sql files applied in name order by pg.Migrate.
//
//go:embed *.sql
var FS embed.FS
