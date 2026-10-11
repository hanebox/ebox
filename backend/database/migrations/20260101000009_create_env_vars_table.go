package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000009CreateEnvVarsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000009CreateEnvVarsTable) Signature() string {
	return "20260101000009_create_env_vars_table"
}

// Up Run the migrations.
func (r *M20260101000009CreateEnvVarsTable) Up() error {
	if facades.Schema().HasTable("env_vars") {
		return nil
	}

	return facades.Schema().Create("env_vars", func(table schema.Blueprint) {
		table.ID()
		table.String("resource_type", 32)
		table.UnsignedBigInteger("resource_id")
		table.String("key")
		table.Text("value")
		table.Boolean("is_build_time").Default(false)
		table.Timestamps()
		table.Index("resource_type")
		table.Index("resource_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000009CreateEnvVarsTable) Down() error {
	return facades.Schema().DropIfExists("env_vars")
}
