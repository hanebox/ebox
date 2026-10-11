package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000012CreateServicesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000012CreateServicesTable) Signature() string {
	return "20260101000012_create_services_table"
}

// Up Run the migrations.
func (r *M20260101000012CreateServicesTable) Up() error {
	if facades.Schema().HasTable("services") {
		return nil
	}

	return facades.Schema().Create("services", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.UnsignedBigInteger("project_id")
		table.UnsignedBigInteger("source_id").Nullable()
		table.String("name")
		table.String("source", 16).Default("inline")
		table.String("repository", 512).Default("")
		table.String("branch").Default("main")
		table.String("compose_path", 512).Default("")
		table.Text("compose").Nullable()
		table.Text("environment").Nullable()
		table.String("entry_service").Default("")
		table.Integer("entry_port").Default(0)
		table.String("status", 32).Default("idle")
		table.Text("last_error").Nullable()
		table.Timestamp("last_deployed_at").Nullable()
		table.Timestamps()
		table.Unique("uuid")
		table.Index("project_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000012CreateServicesTable) Down() error {
	return facades.Schema().DropIfExists("services")
}
