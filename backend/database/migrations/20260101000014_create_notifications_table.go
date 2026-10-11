package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000014CreateNotificationsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000014CreateNotificationsTable) Signature() string {
	return "20260101000014_create_notifications_table"
}

// Up Run the migrations.
func (r *M20260101000014CreateNotificationsTable) Up() error {
	if facades.Schema().HasTable("notification_channels") {
		return nil
	}

	if err := facades.Schema().Create("notification_channels", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.String("name")
		table.String("type", 32)
		table.Boolean("enabled").Default(true)
		table.Text("config").Nullable()
		table.Text("events").Nullable()
		table.Timestamps()
		table.Unique("uuid")
	}); err != nil {
		return err
	}

	return facades.Schema().Create("notification_deliveries", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("channel_id")
		table.String("channel")
		table.String("event", 64)
		table.String("title", 512)
		table.Text("body").Nullable()
		table.String("status", 32)
		table.Text("error").Nullable()
		table.Integer("attempts").Default(0)
		table.Timestamps()
		table.Index("channel_id")
		table.Index("event")
		table.Index("status")
	})
}

// Down Reverse the migrations.
func (r *M20260101000014CreateNotificationsTable) Down() error {
	if err := facades.Schema().DropIfExists("notification_deliveries"); err != nil {
		return err
	}

	return facades.Schema().DropIfExists("notification_channels")
}
