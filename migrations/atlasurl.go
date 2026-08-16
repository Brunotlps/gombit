package migrations

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/LAA-Software-Engineering/gombit/config"
)

var mysqlTCPPattern = regexp.MustCompile(`^(?:([^:@]+)?(?::([^@]*))?@)?tcp\(([^)]+)\)/([^?]*)(\?.*)?$`)

// AtlasURL converts a Gombit database config into an Atlas --url value.
func AtlasURL(cfg config.DatabaseConfig) (string, error) {
	if err := config.ValidateDatabase(cfg); err != nil {
		return "", err
	}
	dsn := strings.TrimSpace(cfg.DSN)
	switch cfg.Driver {
	case config.DatabaseDriverSQLite:
		return sqliteAtlasURL(dsn)
	case config.DatabaseDriverPostgres:
		return postgresAtlasURL(dsn)
	case config.DatabaseDriverMySQL:
		return mysqlAtlasURL(dsn)
	default:
		return "", fmt.Errorf("migrations: unsupported driver %q", cfg.Driver)
	}
}

func sqliteAtlasURL(dsn string) (string, error) {
	switch {
	case strings.HasPrefix(dsn, "sqlite://"):
		return dsn, nil
	case strings.HasPrefix(dsn, "file:"):
		rest := strings.TrimPrefix(dsn, "file:")
		return "sqlite://" + rest, nil
	default:
		return "sqlite://" + dsn, nil
	}
}

func postgresAtlasURL(dsn string) (string, error) {
	switch {
	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		return dsn, nil
	case strings.Contains(dsn, "="):
		return postgresKeyValueURL(dsn)
	default:
		return "", fmt.Errorf("migrations: unsupported postgres DSN %q", dsn)
	}
}

func postgresKeyValueURL(dsn string) (string, error) {
	values := make(map[string]string)
	for _, part := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", fmt.Errorf("migrations: invalid postgres DSN fragment %q", part)
		}
		values[key] = value
	}
	host := values["host"]
	if host == "" {
		host = "localhost"
	}
	port := values["port"]
	if port == "" {
		port = "5432"
	}
	user := values["user"]
	password := values["password"]
	dbname := values["dbname"]
	if dbname == "" {
		return "", fmt.Errorf("migrations: postgres DSN missing dbname")
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   host + ":" + port,
		Path:   "/" + dbname,
	}
	if user != "" {
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}
	query := url.Values{}
	for key, value := range values {
		switch key {
		case "host", "port", "user", "password", "dbname":
			continue
		default:
			query.Set(key, value)
		}
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func mysqlAtlasURL(dsn string) (string, error) {
	if strings.HasPrefix(dsn, "mysql://") {
		return dsn, nil
	}
	matches := mysqlTCPPattern.FindStringSubmatch(dsn)
	if matches == nil {
		return "", fmt.Errorf("migrations: unsupported mysql DSN %q", dsn)
	}
	user := matches[1]
	password := matches[2]
	host := matches[3]
	dbname := matches[4]
	rawQuery := strings.TrimPrefix(matches[5], "?")

	u := &url.URL{
		Scheme:   "mysql",
		Host:     host,
		Path:     "/" + dbname,
		RawQuery: rawQuery,
	}
	if user != "" {
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}
	return u.String(), nil
}
