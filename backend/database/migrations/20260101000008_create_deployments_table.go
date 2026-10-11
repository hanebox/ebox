package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000008CreateDeploymentsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000008CreateDeploymentsTable) Signature() string {
	return "20260101000008_create_deployments_table"
}

// Up Run the migrations.
func (r *M20260101000008CreateDeploymentsTable) Up() error {
	if facades.Schema().HasTable("deployments") {
		return nil
	}

	return facades.Schema().Create("deployments", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.UnsignedBigInteger("application_id")
		table.String("status", 32)
		table.String("trigger", 16)
		table.String("commit", 64)
		table.String("commit_message", 512)
		table.String("image_tag", 255)
		table.String("container_id", 64)
		table.String("container_name", 128)
		table.Text("log")
		table.Text("error")
		table.DateTimeTz("started_at").Nullable()
		table.DateTimeTz("finished_at").Nullable()
		table.BigInteger("duration_ms").Default(0)
		table.Timestamps()
		table.Unique("uuid")
		table.Index("application_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000008CreateDeploymentsTable) Down() error {
	return facades.Schema().DropIfExists("deployments")
}
