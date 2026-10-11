package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000006CreatePersonalAccessTokensTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000006CreatePersonalAccessTokensTable) Signature() string {
	return "20260101000006_create_personal_access_tokens_table"
}

// Up Run the migrations.
func (r *M20260101000006CreatePersonalAccessTokensTable) Up() error {
	if facades.Schema().HasTable("personal_access_tokens") {
		return nil
	}

	return facades.Schema().Create("personal_access_tokens", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("user_id")
		table.String("name")
		table.String("token_hash", 64)
		table.DateTimeTz("last_used_at").Nullable()
		table.DateTimeTz("expires_at").Nullable()
		table.Timestamps()
		table.Unique("token_hash")
		table.Index("user_id")
	})
}

// Down Reverse the migrations.
func (r *M20260101000006CreatePersonalAccessTokensTable) Down() error {
	return facades.Schema().DropIfExists("personal_access_tokens")
}
