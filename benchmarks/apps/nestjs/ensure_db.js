// Create the app's target database if it does not already exist.
//
// TypeORM's migration:run creates tables, not the database, and this app uses
// its own gombit_bench_nestjs database. Provisioning via
// docker-entrypoint-initdb.d would only run on a fresh Postgres volume — this
// suite's volume already exists — so the `migrate` verb calls this on every
// bring-up: a guarded catalog check plus CREATE DATABASE (which has no
// IF NOT EXISTS), idempotent and never requiring `down -v`.
//
// ADMIN_DATABASE_URL is an existing database on the same server (the
// maintenance connection); TARGET_DB is the database to ensure. If either is
// unset (a hand-provisioned local DB), this is a no-op. Uses pg — already the
// app's database driver — rather than adding a psql client to the image.

const { Client } = require('pg');

async function main() {
  const admin = process.env.ADMIN_DATABASE_URL;
  const target = process.env.TARGET_DB;
  if (!admin || !target) {
    return;
  }

  const client = new Client({ connectionString: admin });
  await client.connect();
  try {
    const { rowCount } = await client.query(
      'SELECT 1 FROM pg_database WHERE datname = $1',
      [target],
    );
    if (rowCount === 0) {
      console.error(`ensure_db: creating database ${target}`);
      // Guarded above; quote the identifier defensively.
      await client.query(`CREATE DATABASE "${target.replace(/"/g, '""')}"`);
    }
  } finally {
    await client.end();
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
