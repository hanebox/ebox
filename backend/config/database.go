package config

import (
	"fmt"
	"path/filepath"

	"github.com/goravel/framework/contracts/database/driver"
	sqlitefacades "github.com/goravel/sqlite/facades"

	"ebox/app/facades"
)

func sqliteDSN() string {
	database := facades.Config().EnvString("DB_DATABASE", filepath.Join(eboxDataDir(), "ebox.db"))

	return fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		database,
	)
}

func init() {
	config := facades.Config()
	config.Add("database", map[string]any{
		// Default database connection name
		"default": config.EnvString("DB_CONNECTION", "sqlite"),

		// Database connections
		"connections": map[string]any{
			"sqlite": map[string]any{
				"database": sqliteDSN(),
				"prefix":   "",
				"singular": false,
				"via": func() (driver.Driver, error) {
					return sqlitefacades.Sqlite("sqlite")
				},
			},
		},

		// Pool configuration
		"pool": map[string]any{
			// SQLite is a single-writer database; a small pool combined with
			// WAL mode and busy_timeout keeps reads concurrent without lock errors.
			"max_idle_conns":    2,
			"max_open_conns":    4,
			"conn_max_idletime": 3600,
			"conn_max_lifetime": 3600,
		},

		// Sets the threshold for slow queries in milliseconds, the slow query will be logged.
		"slow_threshold": 200,

		// Migration Repository Table
		"migrations": map[string]any{
			"table": "migrations",
		},
	})
}
