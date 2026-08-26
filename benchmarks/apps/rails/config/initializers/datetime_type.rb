# benchmarks/docs/schema.md's created_at/updated_at columns are
# TIMESTAMPTZ, not the Rails Postgres adapter's default `timestamp without
# time zone` (verified against a fresh db:migrate before this initializer
# existed) — matching benchmarks/apps/gin-gorm and benchmarks/apps/gombit's
# real migrations. This is the Rails-documented way to opt into timestamptz
# for every t.datetime/t.timestamps column, rather than hand-annotating
# each column in every migration.
ActiveSupport.on_load(:active_record_postgresqladapter) do
  self.datetime_type = :timestamptz
end
