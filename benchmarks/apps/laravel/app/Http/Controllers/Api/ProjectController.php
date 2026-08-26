<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Project;
use App\Support\D10;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

// Canonical /api/projects CRUD API (benchmarks/docs/schema.md), implemented
// with Eloquent. Validation is expressed with Laravel's validator; a failure
// throws ValidationException, which bootstrap/app.php maps to a D10
// validation_error (422). A foreign-key / NOT NULL violation that slips past
// validation (e.g. owner_id referencing no user) surfaces as a QueryException
// and is mapped there by SQLSTATE — this controller does not pre-check owner
// existence, leaning on the FK the way benchmarks/apps/gin-gorm and
// benchmarks/apps/django do (rather than an extra existence SELECT per
// create).
class ProjectController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $page = $this->clampPage((int) $request->query('page', 1));
        $limit = $this->clampLimit((int) $request->query('limit', 20));

        $total = Project::count();
        $offset = ($page - 1) * $limit;

        // with('owner'): eager-load owners in one batched
        // `select * from users where id in (...)`, not a JOIN and not one
        // query per row — 3 queries for a non-empty page (COUNT, the page
        // SELECT, the batched owner SELECT), matching gin-gorm's pinned shape
        // exactly; 2 for an empty page (Eloquent skips the owner query when
        // there are no rows). Verified against real Postgres query logs
        // (ProjectListTest).
        $projects = Project::with('owner')
            ->orderByDesc('id')
            ->offset($offset)
            ->limit($limit)
            ->get();

        return D10::dataMeta(
            $projects->map(fn (Project $p) => $this->serialize($p))->all(),
            ['page' => $page, 'limit' => $limit, 'total' => $total],
        );
    }

    public function show(string $id): JsonResponse
    {
        // findOrFail throws ModelNotFoundException (-> D10 404) for both a
        // nonexistent id and a non-numeric one: Postgres rejects a
        // non-integer bind against a bigint column, but findOrFail's own
        // "no such row" path is reached first for the numeric-but-absent
        // case; the non-numeric case is guarded below so it 404s rather than
        // 500ing on a type-cast QueryException.
        return D10::data($this->serialize($this->findProject($id)));
    }

    public function store(Request $request): JsonResponse
    {
        $data = $request->validate([
            'owner_id' => ['required', 'integer'],
            'name' => ['required', 'string', 'max:255', 'regex:/\S/'],
            // sometimes|string: a present null fails `string` (-> 422),
            // an omitted key is skipped (defaults to "" below), a present
            // string — including "" — is stored verbatim. Matches the
            // null-handling contract benchmarks/apps/rails settled on review
            // (reject present-null, default absent to "").
            'description' => ['sometimes', 'string'],
        ]);

        $project = Project::create([
            'owner_id' => $data['owner_id'],
            'name' => $data['name'],
            'description' => $data['description'] ?? '',
        ]);

        return D10::data($this->serialize($project->load('owner')), 201);
    }

    public function update(Request $request, string $id): JsonResponse
    {
        $project = $this->findProject($id);

        $data = $request->validate([
            // sometimes|required, not just sometimes: Laravel skips the
            // non-implicit regex rule on an empty string, so a PATCH
            // {"name": ""} would otherwise pass validation and blank the
            // name. `required` (an implicit rule) rejects "" and null when
            // the key is present; `sometimes` still lets an omitted key
            // through untouched. name absent -> unchanged; name present must
            // be a non-blank string.
            'name' => ['sometimes', 'required', 'string', 'max:255', 'regex:/\S/'],
            'description' => ['sometimes', 'string'],
        ]);

        // Only apply keys the client actually sent (validated above); an
        // absent key leaves the column unchanged, a present one is applied
        // verbatim — symmetric with store.
        if (array_key_exists('name', $data)) {
            $project->name = $data['name'];
        }
        if (array_key_exists('description', $data)) {
            $project->description = $data['description'];
        }
        $project->save();

        return D10::data($this->serialize($project->load('owner')));
    }

    public function destroy(string $id): JsonResponse
    {
        $this->findProject($id)->delete();
        return D10::data(['ok' => true]);
    }

    private function findProject(string $id): Project
    {
        // A route parameter that isn't a positive integer must get the D10
        // not_found envelope, not a raw Postgres type-cast QueryException 500
        // — the same case benchmarks/apps/django's urls.py <str:pk> and
        // benchmarks/apps/rails's find_project! guard handle.
        if (! preg_match('/^\d+$/', $id)) {
            throw new \Illuminate\Database\Eloquent\ModelNotFoundException();
        }
        return Project::with('owner')->findOrFail($id);
    }

    private function serialize(Project $project): array
    {
        return [
            'id' => $project->id,
            'owner_id' => $project->owner_id,
            'owner_name' => $project->owner->name,
            'name' => $project->name,
            'description' => $project->description,
            'created_at' => $project->created_at?->utc()->format('Y-m-d\TH:i:s.u\Z'),
            'updated_at' => $project->updated_at?->utc()->format('Y-m-d\TH:i:s.u\Z'),
        ];
    }

    private function clampPage(int $page): int
    {
        return $page < 1 ? 1 : $page;
    }

    private function clampLimit(int $limit): int
    {
        if ($limit < 1) {
            return 20;
        }
        return $limit > 100 ? 100 : $limit;
    }
}
