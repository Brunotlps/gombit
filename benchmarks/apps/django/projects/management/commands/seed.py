import os

from django.core.management.base import BaseCommand, CommandError
from django.db import connection

from projects.models import Project, User

# issue #141's recommended initial dataset size
# (benchmarks/docs/schema.md "Seed dataset"), matching
# benchmarks/apps/shared.SeedUserCount/SeedProjectCount exactly — this app
# can't import that Go package, so the numbers are duplicated by hand here
# and must be kept in sync with it manually.
SEED_USER_COUNT = 1000
SEED_PROJECT_COUNT = 100000

# How many rows per INSERT — an implementation detail of this seeder, not
# part of the cross-implementation seed contract (mirrors
# benchmarks/apps/gin-gorm's seedBatchSize).
SEED_BATCH_SIZE = 1000


def user_email(i):
    return f"user-{i:04d}@example.com"


def user_name(i):
    return f"User {i:04d}"


def project_owner_id(i, user_count):
    """1-based owner id for project i, round-robin over 1..user_count, so
    every user owns project_count/user_count projects and two
    implementations' seeded row N are content-identical. Port of
    benchmarks/apps/shared.ProjectOwnerID — same formula, independently
    implemented (that package is Go-only).
    """
    if i < 1 or user_count < 1:
        raise ValueError("i and user_count must be positive")
    return (i - 1) % user_count + 1


def project_name(i):
    return f"Project {i:06d}"


def project_description(i):
    return f"Seeded benchmark project {i:06d}"


def seed_database_n(user_count, project_count):
    """Truncates and repopulates the canonical benchmark dataset at the
    given scale. seed_database() always calls this with the canonical
    SEED_USER_COUNT/SEED_PROJECT_COUNT; tests call it directly with small
    counts so they don't pay for a 100,000-row insert.

    Truncating first (RESTART IDENTITY) makes repeated invocations
    idempotent instead of accumulating duplicate data, and resets the users
    sequence to 1 so project_owner_id's round-robin computation stays
    correct without reading back generated ids — mirrors
    benchmarks/apps/gin-gorm's seedDatabaseN exactly.
    """
    with connection.cursor() as cursor:
        cursor.execute("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE")

    User.objects.bulk_create(
        [User(email=user_email(i), name=user_name(i)) for i in range(1, user_count + 1)],
        batch_size=SEED_BATCH_SIZE,
    )
    Project.objects.bulk_create(
        [
            Project(
                owner_id=project_owner_id(i, user_count),
                name=project_name(i),
                description=project_description(i),
            )
            for i in range(1, project_count + 1)
        ],
        batch_size=SEED_BATCH_SIZE,
    )


# BENCH_SEED_USERS / BENCH_SEED_PROJECTS override the row counts for the CI
# smoke's small deterministic seed (issue #141 §11) — same content formulas,
# fewer rows. Unset/empty means the canonical default; a non-empty value that
# is not a positive integer is a fatal error, never a silent fall back to 100k.
# Same names and semantics as benchmarks/apps/shared.SeedCounts (Go).
def _seed_count(env, default):
    raw = os.environ.get(env, "").strip()
    if raw == "":
        return default
    if not raw.isdigit() or int(raw) < 1:
        raise CommandError(f"{env}={raw!r}: must be a positive integer")
    return int(raw)


def seed_database():
    seed_database_n(
        _seed_count("BENCH_SEED_USERS", SEED_USER_COUNT),
        _seed_count("BENCH_SEED_PROJECTS", SEED_PROJECT_COUNT),
    )


class Command(BaseCommand):
    help = "Seed the deterministic BENCH-1 benchmark dataset (1,000 users, 100,000 projects), truncating first."

    def handle(self, *args, **options):
        seed_database()
        self.stdout.write(self.style.SUCCESS("django: seed complete"))
