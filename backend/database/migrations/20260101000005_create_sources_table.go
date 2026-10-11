package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000005CreateSourcesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000005CreateSourcesTable) Signature() string {
	return "20260101000005_create_sources_table"
}

// Up Run the migrations.
func (r *M20260101000005CreateSourcesTable) Up() error {
	if facades.Schema().HasTable("sources") {
		return nil
	}

	return facades.Schema().Create("sources", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.String("name")
		table.String("type", 32)

		table.BigInteger("app_id")
		table.String("app_slug")
		table.String("installation_id", 32)
		table.String("html_url", 512)

		table.String("git_url", 512)
		table.String("auth_type", 16)

		table.Text("private_key_encrypted")
		table.Text("webhook_secret_encrypted")
		table.Text("token_encrypted")
		table.Text("ssh_key_encrypted")

		table.Timestamps()
		table.Unique("uuid")
	})
}

// Down Reverse the migrations.
func (r *M20260101000005CreateSourcesTable) Down() error {
	return facades.Schema().DropIfExists("sources")
}
