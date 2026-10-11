package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/app/facades"
)

type M20260101000013AddComposeProjectToDeployments struct{}

// Signature The unique signature for the migration.
func (r *M20260101000013AddComposeProjectToDeployments) Signature() string {
	return "20260101000013_add_compose_project_to_deployments"
}

// Up Run the migrations.
func (r *M20260101000013AddComposeProjectToDeployments) Up() error {
	if !facades.Schema().HasTable("deployments") {
		return nil
	}

	return facades.Schema().Table("deployments", func(table schema.Blueprint) {
		table.String("compose_project", 128).Default("")
	})
}

// Down Reverse the migrations.
func (r *M20260101000013AddComposeProjectToDeployments) Down() error {
	if !facades.Schema().HasTable("deployments") {
		return nil
	}

	return facades.Schema().Table("deployments", func(table schema.Blueprint) {
		table.DropColumn("compose_project")
	})
}
