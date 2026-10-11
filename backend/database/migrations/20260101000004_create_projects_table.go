package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000004CreateProjectsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000004CreateProjectsTable) Signature() string {
	return "20260101000004_create_projects_table"
}

// Up Run the migrations.
func (r *M20260101000004CreateProjectsTable) Up() error {
	if facades.Schema().HasTable("projects") {
		return nil
	}

	return facades.Schema().Create("projects", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.String("name")
		table.Text("description")
		table.Timestamps()
		table.Unique("uuid")
	})
}

// Down Reverse the migrations.
func (r *M20260101000004CreateProjectsTable) Down() error {
	return facades.Schema().DropIfExists("projects")
}
