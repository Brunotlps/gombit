<?php
// Create the app's target database if it does not already exist.
//
// `php artisan migrate` creates tables, not the database, and this app uses its
// own gombit_bench_laravel database. Provisioning via docker-entrypoint-initdb.d
// would only run on a fresh Postgres volume — this suite's volume already exists
// — so the `migrate` verb calls this on every bring-up: a guarded catalog check
// plus CREATE DATABASE (which has no IF NOT EXISTS), idempotent, no `down -v`.
//
// ADMIN_DATABASE_URL is an existing database on the same server (the maintenance
// connection); TARGET_DB is the database to ensure. If either is unset (a
// hand-provisioned local DB), this is a no-op. Uses PDO/pdo_pgsql — already the
// app's database driver — rather than adding a psql client to the image.

$admin = getenv('ADMIN_DATABASE_URL');
$target = getenv('TARGET_DB');
if ($admin === false || $admin === '' || $target === false || $target === '') {
    exit(0);
}

$parts = parse_url($admin);
if ($parts === false || !isset($parts['host'])) {
    fwrite(STDERR, "ensure_db: could not parse ADMIN_DATABASE_URL\n");
    exit(1);
}
$dsn = sprintf(
    'pgsql:host=%s;port=%d;dbname=%s',
    $parts['host'],
    $parts['port'] ?? 5432,
    ltrim($parts['path'] ?? '', '/')
);

$pdo = new PDO($dsn, $parts['user'] ?? null, $parts['pass'] ?? null, [
    PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
]);

$stmt = $pdo->prepare('SELECT 1 FROM pg_database WHERE datname = ?');
$stmt->execute([$target]);
if ($stmt->fetchColumn() === false) {
    fwrite(STDERR, "ensure_db: creating database {$target}\n");
    // Guarded above; quote the identifier defensively.
    $pdo->exec('CREATE DATABASE "' . str_replace('"', '""', $target) . '"');
}
