package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000015CreateBackupsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260101000015CreateBackupsTable) Signature() string {
	return "20260101000015_create_backups_table"
}

// Up Run the migrations.
func (r *M20260101000015CreateBackupsTable) Up() error {
	if facades.Schema().HasTable("backups") {
		return nil
	}

	if err := facades.Schema().Create("backups", func(table schema.Blueprint) {
		table.ID()
		table.String("uuid", 36)
		table.UnsignedBigInteger("project_id").Nullable()
		table.String("name")
		table.String("target", 16)
		table.String("resource_id", 64).Default("")
		table.String("volume_name")
		table.String("schedule", 64)
		table.Integer("retention").Default(7)
		table.Boolean("enabled").Default(true)
		table.Boolean("stop_during_backup").Default(false)
		table.Timestamp("last_run_at").Nullable()
		table.String("last_status", 32).Default("")
		table.Text("last_error").Nullable()
		table.Timestamps()
		table.Unique("uuid")
		table.Index("project_id")
	}); err != nil {
		return err
	}

	return facades.Schema().Create("backup_runs", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("backup_id")
		table.String("status", 32)
		table.String("trigger", 16)
		table.String("file", 512).Default("")
		table.BigInteger("size_bytes").Default(0)
		table.BigInteger("duration_ms").Default(0)
		table.Text("error").Nullable()
		table.Timestamp("started_at").Nullable()
		table.Timestamp("finished_at").Nullable()
		table.Timestamps()
		table.Index("backup_id")
		table.Index("status")
	})
}

// Down Reverse the migrations.
func (r *M20260101000015CreateBackupsTable) Down() error {
	if err := facades.Schema().DropIfExists("backup_runs"); err != nil {
		return err
	}

	return facades.Schema().DropIfExists("backups")
}
