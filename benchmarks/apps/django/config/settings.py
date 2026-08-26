"""Settings for the BENCH-1 Django+DRF ecosystem-context app (issue #141).

Same canonical /api/projects contract as benchmarks/apps/gin-gorm and
benchmarks/apps/gombit (benchmarks/docs/schema.md), implemented idiomatically
with Django + Django REST Framework so the suite has non-Go ecosystem
context, not just a framework-tax control within Go.

Production configuration (issue #141 §17: "Every application must run in an
explicitly documented production configuration"):

    DEBUG=False (the default below; set DJANGO_DEBUG=true only for local
    development, never for a benchmark run)
    gunicorn as the WSGI server (see README.md) — never `manage.py runserver`
"""
import os
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

BASE_DIR = Path(__file__).resolve().parent.parent


def _env_bool(key, default):
    value = os.environ.get(key)
    if value is None:
        return default
    return value.strip().lower() in ("1", "true", "yes", "on")


def _env_int(key, default):
    value = os.environ.get(key)
    if not value:
        return default
    try:
        return int(value)
    except ValueError:
        return default


# SECURITY WARNING: this key only needs to exist because Django's checks
# require SECRET_KEY to be set; this app has no sessions, no CSRF-protected
# forms, and no signed cookies, so the key is never actually used to protect
# anything. Not a production secret to safeguard — a fixed value keeps
# benchmark runs reproducible without a secrets file to manage.
SECRET_KEY = os.environ.get(
    "DJANGO_SECRET_KEY", "bench-1-django-app-not-a-real-secret"
)

DEBUG = _env_bool("DJANGO_DEBUG", False)

ALLOWED_HOSTS = [
    host.strip()
    for host in os.environ.get("DJANGO_ALLOWED_HOSTS", "127.0.0.1,localhost").split(",")
    if host.strip()
]

INSTALLED_APPS = [
    # django.contrib.auth is required even though this app has no login
    # routes and no AUTH_USER_MODEL of its own: DRF's request.user handling
    # (APIView.initial -> perform_authentication) unconditionally imports
    # django.contrib.auth.models, which raises at import time if that app
    # isn't registered (Permission "doesn't declare an explicit app_label").
    # contenttypes is auth's own dependency. This does add auth's own
    # bookkeeping tables (auth_permission, auth_group, ...) beyond the
    # canonical users/projects schema — harmless extras, the same kind
    # Atlas's own revision-tracking table already is for the Gombit app.
    "django.contrib.contenttypes",
    "django.contrib.auth",
    "rest_framework",
    "projects",
]

MIDDLEWARE = [
    "django.middleware.common.CommonMiddleware",
]

ROOT_URLCONF = "config.urls"
WSGI_APPLICATION = "config.wsgi.application"

USE_TZ = True
TIME_ZONE = "UTC"

DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"

# issue #141 §18 "Connection pooling" pins max open / max idle connections
# (20/20 here, matching benchmarks/apps/gin-gorm and benchmarks/apps/gombit)
# for a single-process pool. gunicorn's pre-fork model means each worker is
# its own process with its own psycopg connection pool (README.md "Database
# connection pooling" explains why a single global pool isn't possible under
# a pre-fork WSGI server) — POOL_MAX_OPEN is therefore divided across
# GUNICORN_WORKERS so the *total* ceiling across all workers still matches
# the pinned 20, not 20-per-worker.
POOL_MAX_OPEN = _env_int("POOL_MAX_OPEN", 20)
GUNICORN_WORKERS = _env_int("GUNICORN_WORKERS", 4)
_PER_WORKER_POOL_SIZE = max(1, POOL_MAX_OPEN // GUNICORN_WORKERS)


def _database_config_from_url(url):
    parts = urlsplit(url)
    query = parse_qs(parts.query)
    sslmode = query.get("sslmode", ["prefer"])[0]

    return {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": parts.path.lstrip("/"),
        "USER": parts.username or "",
        "PASSWORD": parts.password or "",
        "HOST": parts.hostname or "",
        "PORT": str(parts.port or ""),
        # CONN_MAX_AGE stays 0 (Django's default): the psycopg pool below
        # manages connection lifetime, and Django's own docs say not to
        # combine persistent connections with a psycopg pool.
        "CONN_MAX_AGE": 0,
        "OPTIONS": {
            "sslmode": sslmode,
            "pool": {
                "min_size": _PER_WORKER_POOL_SIZE,
                "max_size": _PER_WORKER_POOL_SIZE,
            },
        },
    }


DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_django?sslmode=disable",
)
DATABASES = {"default": _database_config_from_url(DATABASE_URL)}

REST_FRAMEWORK = {
    # This app builds its own D10 envelope by hand in projects/envelope.py
    # (benchmarks/docs/schema.md — Django can't import
    # benchmarks/apps/shared, which is Go-only) rather than DRF's default
    # renderer/pagination classes, so no DEFAULT_PAGINATION_CLASS or
    # DEFAULT_RENDERER_CLASSES override is set here.
    "EXCEPTION_HANDLER": "projects.envelope.exception_handler",
    # No auth on this benchmark's routes at all (issue #141: "No
    # cross-framework auth comparison" for the CRUD apps — Gombit-only auth
    # overhead is Phase 5's own benchmark), matching gin-gorm/gombit having
    # no auth middleware in front of /api/projects either.
    "DEFAULT_AUTHENTICATION_CLASSES": [],
    "DEFAULT_PERMISSION_CLASSES": ["rest_framework.permissions.AllowAny"],
}
