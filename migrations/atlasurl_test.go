package migrations

import (
	"testing"

	"github.com/LAA-Software-Engineering/gombit/config"
)

func TestAtlasURL(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		want    string
		wantErr bool
	}{
		{
			name: "sqlite file",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverSQLite,
				DSN:    "file:gombit.db?cache=shared&_fk=1",
			},
			want: "sqlite://gombit.db?cache=shared&_fk=1",
		},
		{
			name: "sqlite already atlas",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverSQLite,
				DSN:    "sqlite://file?mode=memory&_fk=1",
			},
			want: "sqlite://file?mode=memory&_fk=1",
		},
		{
			name: "postgres url",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverPostgres,
				DSN:    "postgres://gombit:gombit@127.0.0.1:5432/gombit?sslmode=disable",
			},
			want: "postgres://gombit:gombit@127.0.0.1:5432/gombit?sslmode=disable",
		},
		{
			name: "postgres key value",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverPostgres,
				DSN:    "host=localhost user=gombit password=secret dbname=app sslmode=disable",
			},
			want: "postgres://gombit:secret@localhost:5432/app?sslmode=disable",
		},
		{
			name: "mysql tcp",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverMySQL,
				DSN:    "gombit:gombit@tcp(127.0.0.1:3306)/gombit?parseTime=true",
			},
			want: "mysql://gombit:gombit@127.0.0.1:3306/gombit?parseTime=true",
		},
		{
			name: "mysql already atlas",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverMySQL,
				DSN:    "mysql://gombit:gombit@127.0.0.1:3306/gombit",
			},
			want: "mysql://gombit:gombit@127.0.0.1:3306/gombit",
		},
		{
			name: "mysql unsupported",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverMySQL,
				DSN:    "unix(/tmp/mysql.sock)/gombit",
			},
			wantErr: true,
		},
		{
			name: "empty dsn",
			cfg: config.DatabaseConfig{
				Driver: config.DatabaseDriverSQLite,
				DSN:    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AtlasURL(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("AtlasURL() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("AtlasURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("AtlasURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
