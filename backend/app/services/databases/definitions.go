package databases

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ebox/app/models"
	"ebox/app/services/docker"
)

// Definition describes how a database engine runs as a container.
type Definition struct {
	Type         string              `json:"type"`
	Label        string              `json:"label"`
	Image        string              `json:"image"`
	Versions     []string            `json:"versions"`
	InternalPort int                 `json:"internal_port"`
	VolumePath   string              `json:"volume_path"`
	Healthcheck  *docker.Healthcheck `json:"-"`
}

var definitions = map[string]Definition{
	models.DatabaseTypePostgres: {
		Type:         models.DatabaseTypePostgres,
		Label:        "PostgreSQL",
		Image:        "postgres",
		Versions:     []string{"17", "16", "15", "14"},
		InternalPort: 5432,
		VolumePath:   "/var/lib/postgresql/data",
		Healthcheck: &docker.Healthcheck{
			Test:        []string{"CMD-SHELL", "pg_isready -U $POSTGRES_USER -d $POSTGRES_DB"},
			Interval:    5 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 10 * time.Second,
			Retries:     12,
		},
	},
	models.DatabaseTypeMySQL: {
		Type:         models.DatabaseTypeMySQL,
		Label:        "MySQL",
		Image:        "mysql",
		Versions:     []string{"8.4", "8.0"},
		InternalPort: 3306,
		VolumePath:   "/var/lib/mysql",
		Healthcheck: &docker.Healthcheck{
			Test:        []string{"CMD-SHELL", "mysqladmin ping -h 127.0.0.1 -u root --password=\"$MYSQL_ROOT_PASSWORD\" --silent"},
			Interval:    5 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 20 * time.Second,
			Retries:     20,
		},
	},
	models.DatabaseTypeMariaDB: {
		Type:         models.DatabaseTypeMariaDB,
		Label:        "MariaDB",
		Image:        "mariadb",
		Versions:     []string{"11.4", "11", "10.11"},
		InternalPort: 3306,
		VolumePath:   "/var/lib/mysql",
		Healthcheck: &docker.Healthcheck{
			Test:        []string{"CMD", "healthcheck.sh", "--connect", "--innodb_initialized"},
			Interval:    5 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 20 * time.Second,
			Retries:     20,
		},
	},
	models.DatabaseTypeRedis: {
		Type:         models.DatabaseTypeRedis,
		Label:        "Redis",
		Image:        "redis",
		Versions:     []string{"7", "6"},
		InternalPort: 6379,
		VolumePath:   "/data",
		Healthcheck: &docker.Healthcheck{
			Test:        []string{"CMD-SHELL", "redis-cli -a \"$REDIS_PASSWORD\" ping | grep -q PONG"},
			Interval:    5 * time.Second,
			Timeout:     5 * time.Second,
			StartPeriod: 5 * time.Second,
			Retries:     12,
		},
	},
	models.DatabaseTypeMongo: {
		Type:         models.DatabaseTypeMongo,
		Label:        "MongoDB",
		Image:        "mongo",
		Versions:     []string{"7", "6"},
		InternalPort: 27017,
		VolumePath:   "/data/db",
		Healthcheck: &docker.Healthcheck{
			Test:        []string{"CMD-SHELL", "mongosh --quiet --eval \"db.adminCommand('ping')\" || exit 1"},
			Interval:    10 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 20 * time.Second,
			Retries:     12,
		},
	},
}

// Definitions returns every supported engine in display order.
func Definitions() []Definition {
	order := []string{
		models.DatabaseTypePostgres,
		models.DatabaseTypeMySQL,
		models.DatabaseTypeMariaDB,
		models.DatabaseTypeRedis,
		models.DatabaseTypeMongo,
	}

	list := make([]Definition, 0, len(order))
	for _, key := range order {
		list = append(list, definitions[key])
	}

	return list
}

// DefinitionFor returns the engine definition for a database type.
func DefinitionFor(databaseType string) (Definition, error) {
	definition, ok := definitions[databaseType]
	if !ok {
		return Definition{}, fmt.Errorf("databases: unsupported engine %q", databaseType)
	}

	return definition, nil
}

// ImageRef builds the image reference for a version.
func (d Definition) ImageRef(version string) string {
	if version == "" {
		version = d.Versions[0]
	}

	if d.Type == models.DatabaseTypePostgres || d.Type == models.DatabaseTypeRedis {
		return d.Image + ":" + version + "-alpine"
	}

	return d.Image + ":" + version
}

// Environment renders the engine specific container environment.
func (d Definition) Environment(credentials models.DatabaseCredentials) []string {
	switch d.Type {
	case models.DatabaseTypePostgres:
		return []string{
			"POSTGRES_USER=" + credentials.Username,
			"POSTGRES_PASSWORD=" + credentials.Password,
			"POSTGRES_DB=" + credentials.Database,
			"PGDATA=" + d.VolumePath + "/pgdata",
		}
	case models.DatabaseTypeMySQL, models.DatabaseTypeMariaDB:
		return []string{
			"MYSQL_ROOT_PASSWORD=" + credentials.Password,
			"MYSQL_DATABASE=" + credentials.Database,
			"MYSQL_USER=" + credentials.Username,
			"MYSQL_PASSWORD=" + credentials.Password,
		}
	case models.DatabaseTypeRedis:
		return []string{
			"REDIS_PASSWORD=" + credentials.Password,
		}
	case models.DatabaseTypeMongo:
		return []string{
			"MONGO_INITDB_ROOT_USERNAME=" + credentials.Username,
			"MONGO_INITDB_ROOT_PASSWORD=" + credentials.Password,
			"MONGO_INITDB_DATABASE=" + credentials.Database,
		}
	}

	return nil
}

// Command renders the container command, used by engines configured through
// arguments instead of the environment.
func (d Definition) Command(credentials models.DatabaseCredentials) []string {
	switch d.Type {
	case models.DatabaseTypeRedis:
		return []string{"redis-server", "--requirepass", credentials.Password, "--appendonly", "yes"}
	}

	return nil
}

// URL renders a connection string for the engine.
func (d Definition) URL(host string, port int, credentials models.DatabaseCredentials, public bool) string {
	address := host + ":" + strconv.Itoa(port)

	switch d.Type {
	case models.DatabaseTypePostgres:
		return fmt.Sprintf("postgresql://%s:%s@%s/%s", credentials.Username, credentials.Password, address, credentials.Database)
	case models.DatabaseTypeMySQL, models.DatabaseTypeMariaDB:
		return fmt.Sprintf("mysql://%s:%s@%s/%s", credentials.Username, credentials.Password, address, credentials.Database)
	case models.DatabaseTypeRedis:
		return fmt.Sprintf("redis://default:%s@%s/0", credentials.Password, address)
	case models.DatabaseTypeMongo:
		return fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=admin", credentials.Username, credentials.Password, address, credentials.Database)
	}

	return ""
}

// EnvironmentVariables returns ready-to-use environment variables applications
// can consume to reach this database.
func (d Definition) EnvironmentVariables(host string, port int, credentials models.DatabaseCredentials) map[string]string {
	url := d.URL(host, port, credentials, false)

	switch d.Type {
	case models.DatabaseTypePostgres:
		return map[string]string{
			"DB_CONNECTION": "pgsql",
			"DB_HOST":       host,
			"DB_PORT":       strconv.Itoa(port),
			"DB_DATABASE":   credentials.Database,
			"DB_USERNAME":   credentials.Username,
			"DB_PASSWORD":   credentials.Password,
			"DATABASE_URL":  url,
		}
	case models.DatabaseTypeMySQL, models.DatabaseTypeMariaDB:
		return map[string]string{
			"DB_CONNECTION": "mysql",
			"DB_HOST":       host,
			"DB_PORT":       strconv.Itoa(port),
			"DB_DATABASE":   credentials.Database,
			"DB_USERNAME":   credentials.Username,
			"DB_PASSWORD":   credentials.Password,
			"DATABASE_URL":  url,
		}
	case models.DatabaseTypeRedis:
		return map[string]string{
			"REDIS_HOST":     host,
			"REDIS_PORT":     strconv.Itoa(port),
			"REDIS_PASSWORD": credentials.Password,
			"REDIS_URL":      url,
		}
	case models.DatabaseTypeMongo:
		return map[string]string{
			"MONGO_HOST":     host,
			"MONGO_PORT":     strconv.Itoa(port),
			"MONGO_DATABASE": credentials.Database,
			"MONGO_USERNAME": credentials.Username,
			"MONGO_PASSWORD": credentials.Password,
			"MONGODB_URL":    url,
		}
	}

	return map[string]string{}
}

// DefaultVersion returns the preferred version for an engine.
func (d Definition) DefaultVersion() string {
	if len(d.Versions) == 0 {
		return "latest"
	}

	return d.Versions[0]
}

// Username returns the default superuser for an engine.
func (d Definition) Username() string {
	switch d.Type {
	case models.DatabaseTypeRedis:
		return "default"
	case models.DatabaseTypeMongo:
		return "root"
	}

	return "ebox"
}

// NormalizeDatabaseName keeps generated names valid for every engine.
func NormalizeDatabaseName(name string) string {
	normalized := strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z',
			character >= '0' && character <= '9',
			character == '_':
			return character
		case character >= 'A' && character <= 'Z':
			return character + ('a' - 'A')
		}

		return '_'
	}, name)

	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "ebox"
	}

	if normalized[0] >= '0' && normalized[0] <= '9' {
		normalized = "db_" + normalized
	}

	return normalized
}
