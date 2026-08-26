<?php

namespace App\Support;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

// The D10 response envelope (benchmarks/docs/schema.md), reimplemented by
// hand for Laravel — this app can't import benchmarks/apps/shared (Go-only)
// or reuse the Django/Rails envelopes, so it matches the documented JSON
// shape independently, the same way every other benchmarks/apps/
// implementation does for its own language.
//
// Success: {"data": ...} for a single resource, {"data": [...], "meta": {...}}
// for the list endpoint. Error: {"error": {"code", "message", "fields"?}}.
class D10
{
    // Whether this request should get a JSON D10 envelope: every /api/* route
    // (mirrors bootstrap/app.php's original shouldRenderJsonWhen predicate).
    public static function wants(Request $request): bool
    {
        return $request->is('api/*') || $request->expectsJson();
    }

    public static function data(mixed $data, int $status = 200): JsonResponse
    {
        return response()->json(['data' => $data], $status);
    }

    public static function dataMeta(mixed $data, array $meta): JsonResponse
    {
        return response()->json(['data' => $data, 'meta' => $meta], 200);
    }

    public static function error(string $code, string $message, int $status, ?array $fields = null): JsonResponse
    {
        $error = ['code' => $code, 'message' => $message];
        if (! empty($fields)) {
            $error['fields'] = $fields;
        }
        return response()->json(['error' => $error], $status);
    }
}
