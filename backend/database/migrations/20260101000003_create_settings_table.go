package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000003CreateSettingsTable struct{}

func (r *M20260101000003CreateSettingsTable) Signature() string {
	return "20260101000003_create_settings_table"
}

func (r *M20260101000003CreateSettingsTable) Up() error {
	if facades.Schema().HasTable("settings") {
		return nil
	}

	return facades.Schema().Create("settings", func(table schema.Blueprint) {
		table.ID()
		table.String("key", 128)
		table.Text("value")
		table.Timestamps()
		table.Unique("key")
	})
}

func (r *M20260101000003CreateSettingsTable) Down() error {
	return facades.Schema().DropIfExists("settings")
}
