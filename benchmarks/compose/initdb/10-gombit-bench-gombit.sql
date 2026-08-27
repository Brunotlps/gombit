-- Create the second benchmark database the gombit app uses.
--
-- gin-gorm uses gombit_bench (the postgres service's POSTGRES_DB). The gombit
-- app's users/projects tables come from a real Atlas migration, not
-- AutoMigrate, so it must not share a database with gin-gorm (see
-- benchmarks/apps/gombit/README.md). This runs once, on a fresh Postgres data
-- volume, via /docker-entrypoint-initdb.d; `docker compose down -v` resets the
-- volume so it runs again.
CREATE DATABASE gombit_bench_gombit;
