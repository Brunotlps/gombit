import psycopg.errors
from django.db import IntegrityError
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
            project = Project.objects.create(
                owner_id=serializer.validated_data["owner_id"],
                name=serializer.validated_data["name"],
                description=serializer.validated_data.get("description", ""),
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
            project.save()
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
