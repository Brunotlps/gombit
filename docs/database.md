# Database

M1-4 introduces the runtime database boundary. Gombit opens GORM directly and
keeps `*gorm.DB` reachable; it does not define a second ORM interface.

Migration generation is tracked separately in
[`docs/migrations.md`](migrations.md) and
[`docs/adr/012-migrations-atlas-gorm-provider.md`](adr/012-migrations-atlas-gorm-provider.md):
M2 wraps Atlas and `ariga.io/atlas-provider-gorm` rather than defining a
Gombit migration DSL.

## Drivers

`database.Open` supports:

| Driver | Config value |
| --- | --- |
| SQLite | `sqlite` |
| PostgreSQL | `postgres` |
| MySQL | `mysql` |

The opened handle embeds `*gorm.DB` and exposes driver metadata:

```go
db, err := database.Open(cfg.Database)
if err != nil {
	return err
}
defer db.Close()

fmt.Println(db.Driver())
fmt.Println(db.Capabilities().Returning)
```

`framework.App` can receive an opened handle through `framework.WithDatabase`;
the caller owns opening and closing that handle. `app.Database()` returns the
metadata handle and `app.DB()` returns the raw `*gorm.DB` escape hatch.
HTTP-only apps can omit `WithDatabase`.

## Capabilities

`database.Capabilities` captures driver differences that affect generated code
and migrations:

| Capability | SQLite | PostgreSQL | MySQL |
| --- | --- | --- | --- |
| Transactions | yes | yes | yes |
| Savepoints | yes | yes | yes |
| Foreign key constraints | yes | yes | yes |
| Returning | yes | yes | no |
| Upsert | yes | yes | yes |
| Advisory locks | no | yes | no |
| Concurrent index builds | no | yes | no |

## Pool Defaults

If pool settings are left at zero, `database.Open` applies driver-aware
defaults:

| Driver | Max open | Max idle | Connection max lifetime |
| --- | ---: | ---: | --- |
| SQLite | 1 | 1 | none |
| PostgreSQL | 25 | 5 | 30m |
| MySQL | 25 | 5 | 30m |

Set `Config.Database.MaxOpenConns`, `MaxIdleConns`, or `ConnMaxLifetime` to
override these defaults.

The default SQLite DSN writes `gombit.db` in the current working directory.
Production checks for unwritable SQLite paths are tracked with the later
Appendix C hardening work.

## Integration Tests

The default unit suite exercises SQLite without external services. Postgres and
MySQL conformance can be run with the `integration` build tag and explicit DSN
flags:

```sh
go test -tags integration ./database \
  -database.postgres-dsn 'postgres://gombit:gombit@127.0.0.1:5432/gombit?sslmode=disable' \
  -database.mysql-dsn 'gombit:gombit@tcp(127.0.0.1:3306)/gombit?parseTime=true'
```
