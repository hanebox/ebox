package databases

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ebox/app/models"
)

func TestDefinitionsCoverEveryEngine(t *testing.T) {
	list := Definitions()
	require.Len(t, list, 5)

	types := []string{}
	for _, definition := range list {
		types = append(types, definition.Type)
		require.NotEmpty(t, definition.Versions)
		require.Greater(t, definition.InternalPort, 0)
		require.NotEmpty(t, definition.VolumePath)
		require.NotNil(t, definition.Healthcheck)
		require.NotEmpty(t, definition.Healthcheck.Test)
	}

	require.Equal(t, []string{
		models.DatabaseTypePostgres,
		models.DatabaseTypeMySQL,
		models.DatabaseTypeMariaDB,
		models.DatabaseTypeRedis,
		models.DatabaseTypeMongo,
	}, types)
}

func TestDefinitionForRejectsUnknownEngines(t *testing.T) {
	_, err := DefinitionFor("sqlite")
	require.Error(t, err)
}

func TestImageRefTags(t *testing.T) {
	postgres, err := DefinitionFor(models.DatabaseTypePostgres)
	require.NoError(t, err)
	require.Equal(t, "postgres:17-alpine", postgres.ImageRef("17"))
	require.Equal(t, "postgres:17-alpine", postgres.ImageRef(""))

	redis, err := DefinitionFor(models.DatabaseTypeRedis)
	require.NoError(t, err)
	require.Equal(t, "redis:7-alpine", redis.ImageRef("7"))

	mysql, err := DefinitionFor(models.DatabaseTypeMySQL)
	require.NoError(t, err)
	require.Equal(t, "mysql:8.4", mysql.ImageRef("8.4"))
}

func TestEnvironmentPerEngine(t *testing.T) {
	credentials := models.DatabaseCredentials{Username: "ebox", Password: "s3cret", Database: "app_db"}

	postgres, _ := DefinitionFor(models.DatabaseTypePostgres)
	require.Contains(t, postgres.Environment(credentials), "POSTGRES_PASSWORD=s3cret")
	require.Contains(t, postgres.Environment(credentials), "POSTGRES_DB=app_db")

	mysql, _ := DefinitionFor(models.DatabaseTypeMySQL)
	require.Contains(t, mysql.Environment(credentials), "MYSQL_ROOT_PASSWORD=s3cret")
	require.Contains(t, mysql.Environment(credentials), "MYSQL_USER=ebox")

	mongo, _ := DefinitionFor(models.DatabaseTypeMongo)
	mongoCredentials := models.DatabaseCredentials{Username: mongo.Username(), Password: "s3cret", Database: "app_db"}
	require.Contains(t, mongo.Environment(mongoCredentials), "MONGO_INITDB_ROOT_USERNAME=root")

	redis, _ := DefinitionFor(models.DatabaseTypeRedis)
	require.Equal(t, []string{"redis-server", "--requirepass", "s3cret", "--appendonly", "yes"}, redis.Command(credentials))
}

func TestURLs(t *testing.T) {
	credentials := models.DatabaseCredentials{Username: "ebox", Password: "p@ss:word", Database: "app_db"}

	postgres, _ := DefinitionFor(models.DatabaseTypePostgres)
	require.Equal(t, "postgresql://ebox:p@ss:word@db:5432/app_db", postgres.URL("db", 5432, credentials, false))

	mysql, _ := DefinitionFor(models.DatabaseTypeMySQL)
	require.Equal(t, "mysql://ebox:p@ss:word@db:3306/app_db", mysql.URL("db", 3306, credentials, false))

	redis, _ := DefinitionFor(models.DatabaseTypeRedis)
	require.Equal(t, "redis://default:p@ss:word@db:6379/0", redis.URL("db", 6379, credentials, false))

	mongo, _ := DefinitionFor(models.DatabaseTypeMongo)
	require.Equal(t, "mongodb://ebox:p@ss:word@db:27017/app_db?authSource=admin", mongo.URL("db", 27017, credentials, false))
}

func TestEnvironmentVariablesForAttach(t *testing.T) {
	credentials := models.DatabaseCredentials{Username: "ebox", Password: "secret", Database: "app_db"}

	postgres, _ := DefinitionFor(models.DatabaseTypePostgres)
	values := postgres.EnvironmentVariables("ebox-db-1234", 5432, credentials)
	require.Equal(t, "pgsql", values["DB_CONNECTION"])
	require.Equal(t, "ebox-db-1234", values["DB_HOST"])
	require.Equal(t, "secret", values["DB_PASSWORD"])

	redis, _ := DefinitionFor(models.DatabaseTypeRedis)
	redisValues := redis.EnvironmentVariables("ebox-db-1234", 6379, credentials)
	require.Equal(t, "ebox-db-1234", redisValues["REDIS_HOST"])
	require.Equal(t, "secret", redisValues["REDIS_PASSWORD"])
	require.Contains(t, redisValues["REDIS_URL"], "redis://")
}

func TestUsernames(t *testing.T) {
	redis, _ := DefinitionFor(models.DatabaseTypeRedis)
	require.Equal(t, "default", redis.Username())

	mongo, _ := DefinitionFor(models.DatabaseTypeMongo)
	require.Equal(t, "root", mongo.Username())

	postgres, _ := DefinitionFor(models.DatabaseTypePostgres)
	require.Equal(t, "ebox", postgres.Username())
}

func TestNormalizeDatabaseName(t *testing.T) {
	require.Equal(t, "my_app", NormalizeDatabaseName("My App"))
	require.Equal(t, "app_db", NormalizeDatabaseName("app-db"))
	require.Equal(t, "db_2024", NormalizeDatabaseName("2024"))
	require.Equal(t, "ebox", NormalizeDatabaseName("!!!"))
	require.Equal(t, "a_b", NormalizeDatabaseName("  a b  "))
}
