"""Create the app's target database if it does not already exist.

Django's ``manage.py migrate`` creates tables, not the database, and this app
uses its own ``gombit_bench_django`` database (it must not share another app's
schema). Provisioning it via docker-entrypoint-initdb.d would only run on a
fresh Postgres volume — this suite's volume already exists — so instead the
``migrate`` verb calls this on every bring-up: a guarded catalog check plus
``CREATE DATABASE`` (which has no IF NOT EXISTS), idempotent and never
requiring ``down -v``.

ADMIN_DATABASE_URL is an existing database on the same server (the maintenance
connection); TARGET_DB is the database to ensure. If either is unset (a
hand-provisioned local DB), this is a no-op.
"""

import os
import sys

import psycopg
from psycopg import sql


def main() -> None:
    admin = os.environ.get("ADMIN_DATABASE_URL")
    target = os.environ.get("TARGET_DB")
    if not admin or not target:
        return

    # autocommit: CREATE DATABASE cannot run inside a transaction block.
    with psycopg.connect(admin, autocommit=True) as conn:
        exists = conn.execute(
            "SELECT 1 FROM pg_database WHERE datname = %s", (target,)
        ).fetchone()
        if exists:
            return
        print(f"ensure_db: creating database {target}", file=sys.stderr)
        conn.execute(sql.SQL("CREATE DATABASE {}").format(sql.Identifier(target)))


if __name__ == "__main__":
    main()
