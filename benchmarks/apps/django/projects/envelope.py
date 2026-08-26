"""D10 response envelope (benchmarks/docs/schema.md), reimplemented by hand
for Django — this app can't import benchmarks/apps/shared (Go-only), so it
matches the documented JSON shape independently, the same way
benchmarks/apps/gin-gorm's shared.ErrorEnvelope does for the Go control app.

Success: {"data": ...} for a single resource, {"data": [...], "meta": {...}}
for the list endpoint. Error: {"error": {"code", "message", "fields"?}}.
"""
from rest_framework import status
from rest_framework.response import Response
from rest_framework.views import exception_handler as drf_exception_handler


def success(data, http_status=status.HTTP_200_OK):
    return Response({"data": data}, status=http_status)


def success_with_meta(data, meta):
    return Response({"data": data, "meta": meta})


def error_response(code, message, http_status, fields=None):
    body = {"code": code, "message": message}
    if fields:
        body["fields"] = fields
    return Response({"error": body}, status=http_status)


def not_found(message="not found"):
    return error_response("not_found", message, status.HTTP_404_NOT_FOUND)


def conflict(message):
    return error_response("conflict", message, status.HTTP_409_CONFLICT)


def validation_error(message, fields=None):
    return error_response(
        "validation_error", message, status.HTTP_422_UNPROCESSABLE_ENTITY, fields
    )


def internal(message):
    return error_response("internal", message, status.HTTP_500_INTERNAL_SERVER_ERROR)


def exception_handler(exc, context):
    """Reshapes DRF's own exception responses (malformed JSON body,
    unsupported media type, and anything else rejected before a view method
    runs) into the D10 error envelope, so every error path matches
    benchmarks/docs/schema.md — not just the ones this app's views build
    explicitly via the helpers above.

    D10 fixes the status for each category (not_found -> 404,
    conflict -> 409, validation_error -> 422, internal -> 500); a category
    is chosen by *bucketing* DRF's native status (4xx that isn't 404/409 is
    invalid input, full stop), and the D10 status for that category is what
    gets returned — never DRF's own native status passed through unchanged.
    A DRF ParseError (malformed JSON) is native HTTP 400: relabeling its
    body as "validation_error" while leaving the status at 400 would still
    not match benchmarks/apps/gin-gorm, which maps the equivalent
    ShouldBindJSON failure to 422 — issue #141 §15 requires equivalent
    invalid input to be rejected the same way, and a D10 code with the
    wrong status is not actually D10.
    """
    response = drf_exception_handler(exc, context)
    if response is None:
        # An exception DRF doesn't know how to handle (e.g. an uncaught
        # IntegrityError) — let Django's own DEBUG=False 500 handling take
        # over rather than guess at a D10 shape for an error this app never
        # anticipated.
        return None

    if response.status_code >= 500:
        return internal(str(exc))
    if response.status_code == status.HTTP_404_NOT_FOUND:
        return not_found(str(exc))
    if response.status_code == status.HTTP_409_CONFLICT:
        return conflict(str(exc))
    # Every other 4xx DRF itself rejects a request with (malformed JSON,
    # unsupported media type, method not allowed, throttled, ...) is
    # invalid input by D10's own category mapping — validation_error is
    # always 422, not whatever native 4xx DRF's exception type happened to
    # carry.
    return validation_error(str(exc))
