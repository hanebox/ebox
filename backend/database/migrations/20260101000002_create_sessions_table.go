package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000002CreateSessionsTable struct{}

func (r *M20260101000002CreateSessionsTable) Signature() string {
	return "20260101000002_create_sessions_table"
}

func (r *M20260101000002CreateSessionsTable) Up() error {
	if facades.Schema().HasTable("sessions") {
		return nil
	}

	return facades.Schema().Create("sessions", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("user_id")
		table.String("token_hash", 64)
		table.String("csrf_hash", 64)
		table.String("ip", 64)
		table.String("user_agent", 512)
		table.DateTimeTz("last_seen_at")
		table.DateTimeTz("expires_at")
		table.Timestamps()
		table.Unique("token_hash")
		table.Index("user_id")
		table.Index("expires_at")
	})
}

func (r *M20260101000002CreateSessionsTable) Down() error {
	return facades.Schema().DropIfExists("sessions")
}
