<?php

use App\Http\Controllers\Api\ProjectController;
use Illuminate\Support\Facades\Route;

// bootstrap/app.php's withRouting(api: ...) applies the /api prefix and the
// api middleware group (no session/CSRF), so these register as
// /api/projects. apiResource maps index/store/show/update/destroy to
// GET/POST/GET{id}/PATCH{id}/DELETE{id} — the canonical five routes.
Route::apiResource('projects', ProjectController::class);
