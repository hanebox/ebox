package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000001CreateUsersTable struct{}

func (r *M20260101000001CreateUsersTable) Signature() string {
	return "20260101000001_create_users_table"
}

func (r *M20260101000001CreateUsersTable) Up() error {
	if facades.Schema().HasTable("users") {
		return nil
	}

	return facades.Schema().Create("users", func(table schema.Blueprint) {
		table.ID()
		table.String("name")
		table.String("email")
		table.String("password_hash")
		table.Timestamps()
		table.Unique("email")
	})
}

func (r *M20260101000001CreateUsersTable) Down() error {
	return facades.Schema().DropIfExists("users")
}
