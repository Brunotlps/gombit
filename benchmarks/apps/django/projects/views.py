import psycopg.errors
from django.db import IntegrityError, connection, transaction
from rest_framework import status
from rest_framework.views import APIView

from .envelope import (
    conflict,
    internal,
    not_found,
    success,
    success_with_meta,
    validation_error,
)
from .models import Project
from .serializers import (
    CreateProjectSerializer,
    ProjectDataSerializer,
    UpdateProjectSerializer,
)


def _clamp_page(page, limit):
    page = max(page, 1)
    if limit < 1:
        limit = 20
    limit = min(limit, 100)
    return page, limit


def _query_int(request, key, default):
    raw = request.query_params.get(key)
    if raw is None:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def _serialize(project):
    return ProjectDataSerializer(project).data


def _parse_pk(pk):
    try:
        return int(pk)
    except (TypeError, ValueError):
        return None


def _map_integrity_error(exc, conflict_message):
    """Maps a GORM/psycopg write error to a D10 category error by Postgres
    SQLSTATE, the same policy benchmarks/apps/gin-gorm's mapPersistError
    uses: a unique-constraint violation becomes conflict, a foreign-key
    violation (e.g. POST with an owner_id that doesn't reference an
    existing user) becomes validation_error — a bad client-supplied
    reference is invalid input, not a server failure (issue #141 §15).
    Anything else becomes internal.
    """
    cause = exc.__cause__
    if isinstance(cause, psycopg.errors.UniqueViolation):
        return conflict(conflict_message)
    if isinstance(cause, psycopg.errors.ForeignKeyViolation):
        return validation_error("request references a resource that does not exist")
    return internal("persist project")


def _write_project(write):
    """Runs `write` (a zero-arg callable that inserts/updates a Project)
    inside its own atomic block and forces Postgres to check deferred
    constraints before that block exits, instead of leaving them pending
    until the enclosing transaction commits.

    Django's Postgres backend always emits foreign keys as DEFERRABLE
    INITIALLY DEFERRED (django/db/backends/postgresql/operations.py
    deferrable_sql — not configurable per-field), unlike
    benchmarks/apps/gin-gorm/gombit's plain, immediately-checked FK. Outside
    any wrapping transaction (a real request under Django's default
    per-statement autocommit) this is invisible: the INSERT's own implicit
    transaction ends immediately after it runs, so Postgres checks the
    deferred constraint right there anyway. But inside any wrapping
    transaction — this app's own test suite (TestCase wraps each test in
    one), or a production deployment with ATOMIC_REQUESTS=True — a
    foreign-key violation would NOT raise at Project.objects.create()/
    .save() at all; the INSERT/UPDATE would silently "succeed" until the
    enclosing transaction finally commits, and the immediate
    select_related(...).get(...) reload right after would instead raise
    Project.DoesNotExist (owner_id doesn't match any row, so the JOIN drops
    it) — an unrelated-looking exception this app has no reason to expect
    from a reload of a row it just wrote, instead of the IntegrityError
    _map_integrity_error is built to classify. Verified by reproducing it:
    removing this wrapper reproduces exactly that DoesNotExist under
    TestCase; restoring it raises IntegrityError as expected. atomic() also
    isolates the failure to a savepoint so catching IntegrityError doesn't
    poison the rest of an enclosing transaction, per Django's own atomic()
    documentation.
    """
    with transaction.atomic():
        result = write()
        connection.check_constraints()
    return result


class ProjectListCreateView(APIView):
    def get(self, request):
        page, limit = _clamp_page(_query_int(request, "page", 1), _query_int(request, "limit", 20))

        total = Project.objects.count()
        offset = (page - 1) * limit
        # select_related("owner") issues a single JOIN for the page —
        # benchmarks/docs/schema.md "Canonical CRUD API" requires a query
        # count independent of page size, not one owner query per row.
        # COUNT + the joined page SELECT is 2 queries total, a stricter
        # (but schema.md-permitted: "any eager-load strategy that keeps the
        # query count independent of page size") shape than gin-gorm's 3
        # (COUNT + page + a separate batched owner IN (...)). Documented
        # here rather than silently diverging while claiming the same
        # count — see ProjectListTests.test_list_does_not_n_plus_1[_empty_page].
        rows = list(
            Project.objects.select_related("owner").order_by("-id")[offset : offset + limit]
        )
        data = [_serialize(row) for row in rows]
        return success_with_meta(data, {"page": page, "limit": limit, "total": total})

    def post(self, request):
        serializer = CreateProjectSerializer(data=request.data)
        if not serializer.is_valid():
            return validation_error("invalid request body", serializer.errors)

        try:
            project = _write_project(
                lambda: Project.objects.create(
                    owner_id=serializer.validated_data["owner_id"],
                    name=serializer.validated_data["name"],
                    description=serializer.validated_data.get("description", ""),
                )
            )
        except IntegrityError as exc:
            return _map_integrity_error(exc, "project already exists")

        project = Project.objects.select_related("owner").get(pk=project.pk)
        return success(_serialize(project), http_status=status.HTTP_201_CREATED)


class ProjectDetailView(APIView):
    def get(self, request, pk):
        project_id = _parse_pk(pk)
        if project_id is None:
            return not_found("project not found")
        try:
            project = Project.objects.select_related("owner").get(pk=project_id)
        except Project.DoesNotExist:
            return not_found("project not found")
        return success(_serialize(project))

    def patch(self, request, pk):
        project_id = _parse_pk(pk)
        if project_id is None:
            return not_found("project not found")

        serializer = UpdateProjectSerializer(data=request.data)
        if not serializer.is_valid():
            return validation_error("invalid request body", serializer.errors)

        try:
            project = Project.objects.get(pk=project_id)
        except Project.DoesNotExist:
            return not_found("project not found")

        if "name" in serializer.validated_data:
            project.name = serializer.validated_data["name"]
        if "description" in serializer.validated_data:
            project.description = serializer.validated_data["description"]

        try:
            _write_project(project.save)
        except IntegrityError as exc:
            return _map_integrity_error(exc, "project already exists")

        project = Project.objects.select_related("owner").get(pk=project.pk)
        return success(_serialize(project))

    def delete(self, request, pk):
        project_id = _parse_pk(pk)
        if project_id is None:
            return not_found("project not found")
        try:
            project = Project.objects.get(pk=project_id)
        except Project.DoesNotExist:
            return not_found("project not found")
        project.delete()
        return success({"ok": True})
