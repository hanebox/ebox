package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000007CreateApplicationsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000007CreateApplicationsTable) Signature() string {
	return "20260101000007_create_applications_table"
}

// Up Run the migrations.
func (r *M20260101000007CreateApplicationsTable) Up() error {
	if facades.Schema().HasTable("applications") {
		return nil
	}

	return facades.Schema().Create("applications", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.UnsignedBigInteger("project_id")
		table.UnsignedBigInteger("source_id").Nullable()
		table.String("name")
		table.String("kind", 16)
		table.String("repository", 512)
		table.String("branch")
		table.String("build_pack", 32)
		table.Text("config")
		table.Boolean("auto_deploy").Default(false)
		table.String("deploy_token", 64)
		table.String("status", 32).Default("idle")
		table.Timestamps()
		table.Unique("uuid")
		table.Unique("deploy_token")
		table.Index("project_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000007CreateApplicationsTable) Down() error {
	return facades.Schema().DropIfExists("applications")
}
