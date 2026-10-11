package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000010CreateDomainsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000010CreateDomainsTable) Signature() string {
	return "20260101000010_create_domains_table"
}

// Up Run the migrations.
func (r *M20260101000010CreateDomainsTable) Up() error {
	if facades.Schema().HasTable("domains") {
		return nil
	}

	return facades.Schema().Create("domains", func(table schema.Blueprint) {
		table.ID()
		table.String("resource_type", 32)
		table.UnsignedBigInteger("resource_id")
		table.String("host")
		table.Integer("port").Default(80)
		table.Boolean("https").Default(true)
		table.Boolean("is_primary").Default(false)
		table.Timestamps()
		table.Unique("host")
		table.Index("resource_type")
		table.Index("resource_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000010CreateDomainsTable) Down() error {
	return facades.Schema().DropIfExists("domains")
}
