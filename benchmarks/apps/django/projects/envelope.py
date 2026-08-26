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


# _STATUS_CODES covers the D10 categories benchmarks/apps/gin-gorm's own
# shared.ErrorEnvelope defines (not_found, conflict, validation_error,
# internal) — the only ones this app's views raise or that DRF's own
# exception handling produces for routes actually exercised by the fairness
# suite. A status this app never produces on purpose (e.g. 405) falls back
# to the nearest bucket by status range rather than inventing a fifth code
# category no other implementation has.
_STATUS_CODES = {
    status.HTTP_404_NOT_FOUND: "not_found",
    status.HTTP_409_CONFLICT: "conflict",
    status.HTTP_422_UNPROCESSABLE_ENTITY: "validation_error",
}


def exception_handler(exc, context):
    """Reshapes DRF's own exception responses (malformed JSON body,
    unsupported media type, and anything else rejected before a view method
    runs) into the D10 error envelope, so every error path matches
    benchmarks/docs/schema.md — not just the ones this app's views build
    explicitly via the helpers above.
    """
    response = drf_exception_handler(exc, context)
    if response is None:
        # An exception DRF doesn't know how to handle (e.g. an uncaught
        # IntegrityError) — let Django's own DEBUG=False 500 handling take
        # over rather than guess at a D10 shape for an error this app never
        # anticipated.
        return None

    if response.status_code >= 500:
        code = "internal"
    else:
        code = _STATUS_CODES.get(response.status_code, "validation_error")
    return Response({"error": {"code": code, "message": str(exc)}}, status=response.status_code)
