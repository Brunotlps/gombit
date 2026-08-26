<?php

use App\Support\D10;
use Illuminate\Database\Eloquent\ModelNotFoundException;
use Illuminate\Database\QueryException;
use Illuminate\Foundation\Application;
use Illuminate\Foundation\Configuration\Exceptions;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Foundation\Http\Middleware\ConvertEmptyStringsToNull;
use Illuminate\Foundation\Http\Middleware\TrimStrings;
use Illuminate\Http\Request;
use Illuminate\Validation\ValidationException;
use Symfony\Component\HttpKernel\Exception\NotFoundHttpException;

return Application::configure(basePath: dirname(__DIR__))
    ->withRouting(
        web: __DIR__.'/../routes/web.php',
        api: __DIR__.'/../routes/api.php',
        commands: __DIR__.'/../routes/console.php',
        // /livez, matching benchmarks/apps/gin-gorm/gombit/rails and what
        // benchmarks/apps/fairness_test.go's waitForHealth polls (Laravel's
        // own default is /up).
        health: '/livez',
    )
    ->withMiddleware(function (Middleware $middleware): void {
        // Laravel applies TrimStrings and ConvertEmptyStringsToNull to every
        // request globally by default. Both silently rewrite client input
        // before any controller sees it: TrimStrings would strip the
        // leading/trailing whitespace the canonical contract requires be
        // stored byte-for-byte (the exact issue benchmarks/apps/django's
        // review caught in its serializer), and ConvertEmptyStringsToNull
        // would turn a legitimate description "" into null (which this app
        // then rejects as invalid) — two framework defaults that would make
        // this implementation silently disagree with gin-gorm/gombit, which
        // store what the client sent. Removed so client strings reach the
        // controller unaltered.
        $middleware->remove([
            TrimStrings::class,
            ConvertEmptyStringsToNull::class,
        ]);
    })
    ->withExceptions(function (Exceptions $exceptions): void {
        // Render every known error path in the D10 envelope
        // (benchmarks/docs/schema.md) for API requests, rather than
        // Laravel's default {"message","errors"} shape.
        $exceptions->render(function (ValidationException $e, Request $request) {
            if (! D10::wants($request)) {
                return null;
            }
            return D10::error('validation_error', 'invalid request body', 422, $e->errors());
        });

        $exceptions->render(function (ModelNotFoundException $e, Request $request) {
            return D10::wants($request) ? D10::error('not_found', 'project not found', 404) : null;
        });

        $exceptions->render(function (NotFoundHttpException $e, Request $request) {
            return D10::wants($request) ? D10::error('not_found', 'project not found', 404) : null;
        });

        // A QueryException wraps the PDOException; its SQLSTATE distinguishes
        // the client-caused constraint violations from a genuine server
        // fault — the same policy benchmarks/apps/gin-gorm's mapPersistError
        // and benchmarks/apps/django's _map_integrity_error use (issue #141
        // §15: equivalent invalid input rejected the same way).
        $exceptions->render(function (QueryException $e, Request $request) {
            if (! D10::wants($request)) {
                return null;
            }
            return match ($e->getCode()) {
                '23505' => D10::error('conflict', 'project already exists', 409),
                '23503' => D10::error('validation_error', 'request references a resource that does not exist', 422),
                '23502' => D10::error('validation_error', 'a required field must not be null', 422),
                default => D10::error('internal', 'database error', 500),
            };
        });
    })->create();
