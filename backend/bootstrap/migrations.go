package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"

	"ebox/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20210101000001CreateJobsTable{},
		&migrations.M20260101000001CreateUsersTable{},
		&migrations.M20260101000002CreateSessionsTable{},
		&migrations.M20260101000003CreateSettingsTable{},
		&migrations.M20260101000004CreateProjectsTable{},
		&migrations.M20260101000005CreateSourcesTable{},
		&migrations.M20260101000006CreatePersonalAccessTokensTable{},
		&migrations.M20260101000007CreateApplicationsTable{},
		&migrations.M20260101000008CreateDeploymentsTable{},
		&migrations.M20260101000009CreateEnvVarsTable{},
		&migrations.M20260101000010CreateDomainsTable{},
		&migrations.M20260101000011CreateDatabasesTable{},
		&migrations.M20260101000012CreateServicesTable{},
		&migrations.M20260101000013AddComposeProjectToDeployments{},
		&migrations.M20260101000014CreateNotificationsTable{},
		&migrations.M20260101000015CreateBackupsTable{},
	}
}
