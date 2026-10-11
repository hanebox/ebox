package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000011CreateDatabasesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000011CreateDatabasesTable) Signature() string {
	return "20260101000011_create_databases_table"
}

// Up Run the migrations.
func (r *M20260101000011CreateDatabasesTable) Up() error {
	if facades.Schema().HasTable("databases") {
		return nil
	}

	return facades.Schema().Create("databases", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.UnsignedBigInteger("project_id")
		table.String("name")
		table.String("type", 32)
		table.String("version", 32)
		table.Integer("port").Default(0)
		table.Text("credentials")
		table.Text("limits").Nullable()
		table.String("status", 32).Default("creating")
		table.Text("status_message").Nullable()
		table.Timestamps()
		table.Unique("uuid")
		table.Index("project_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000011CreateDatabasesTable) Down() error {
	return facades.Schema().DropIfExists("databases")
}
