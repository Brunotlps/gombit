<?php

namespace Tests\Feature;

use App\Models\Project;
use App\Models\User;
use App\Support\CanonicalSeed;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

// Covers the same contract benchmarks/apps/gin-gorm/main_test.go,
// benchmarks/apps/gombit/internal/project/handler_test.go,
// benchmarks/apps/django/projects/tests.py, and
// benchmarks/apps/rails/test/controllers/api/projects_controller_test.rb pin
// for their own implementations (benchmarks/docs/schema.md), so the five
// suites can't silently diverge while each claims the same contract. Asserts
// the D10 error.code, not just the HTTP status, on every rejection — a lesson
// from benchmarks/apps/django's and rails's review rounds.
class ProjectApiTest extends TestCase
{
    use RefreshDatabase;

    private function makeOwner(): User
    {
        return User::create(['email' => 'owner@example.com', 'name' => 'Owner']);
    }

    public function test_crud_round_trip(): void
    {
        $owner = $this->makeOwner();

        $create = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'Test Project', 'description' => 'desc']);
        $create->assertStatus(201);
        $created = $create->json('data');
        $this->assertSame('Test Project', $created['name']);
        $this->assertSame('Owner', $created['owner_name']);

        $this->getJson("/api/projects/{$created['id']}")->assertStatus(200);

        $update = $this->patchJson("/api/projects/{$created['id']}", ['name' => 'Renamed']);
        $update->assertStatus(200);
        $this->assertSame('Renamed', $update->json('data.name'));
        $this->assertSame('desc', $update->json('data.description')); // unchanged

        $this->deleteJson("/api/projects/{$created['id']}")->assertStatus(200);
        $this->getJson("/api/projects/{$created['id']}")->assertStatus(404);
    }

    public function test_rejects_blank_name_on_create(): void
    {
        $owner = $this->makeOwner();
        foreach (['', '   '] as $name) {
            $response = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => $name, 'description' => 'x']);
            $response->assertStatus(422);
            $this->assertSame('validation_error', $response->json('error.code'));
        }
    }

    public function test_rejects_blank_name_on_update(): void
    {
        $owner = $this->makeOwner();
        $id = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'Original', 'description' => 'desc'])->json('data.id');

        foreach (['', '   '] as $name) {
            $response = $this->patchJson("/api/projects/{$id}", ['name' => $name]);
            $response->assertStatus(422);
            $this->assertSame('validation_error', $response->json('error.code'));
        }

        // A rejected update must not have partially applied.
        $this->assertSame('Original', $this->getJson("/api/projects/{$id}")->json('data.name'));
    }

    public function test_rejects_zero_owner_id(): void
    {
        $response = $this->postJson('/api/projects', ['owner_id' => 0, 'name' => 'x']);
        $response->assertStatus(422);
        $this->assertSame('validation_error', $response->json('error.code'));
    }

    public function test_rejects_nonexistent_owner_id(): void
    {
        $response = $this->postJson('/api/projects', ['owner_id' => 999999, 'name' => 'Orphan']);
        $response->assertStatus(422);
        $this->assertSame('validation_error', $response->json('error.code'));
    }

    public function test_rejects_malformed_json(): void
    {
        $response = $this->call('POST', '/api/projects', [], [], [], ['CONTENT_TYPE' => 'application/json', 'HTTP_ACCEPT' => 'application/json'], '{');
        $response->assertStatus(422);
        $this->assertSame('validation_error', $response->json('error.code'));
    }

    public function test_preserves_description_whitespace_and_empty_string(): void
    {
        $owner = $this->makeOwner();

        // Leading/trailing whitespace must survive (Laravel's global
        // TrimStrings middleware is removed in bootstrap/app.php).
        $padded = '  padded  ';
        $create = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'x', 'description' => $padded]);
        $create->assertStatus(201);
        $this->assertSame($padded, $create->json('data.description'));

        // An empty string must stay "" (ConvertEmptyStringsToNull is removed
        // too — otherwise "" would become null and get rejected).
        $empty = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'y', 'description' => '']);
        $empty->assertStatus(201);
        $this->assertSame('', $empty->json('data.description'));
    }

    public function test_create_without_description_defaults_to_empty_string(): void
    {
        $owner = $this->makeOwner();
        $create = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'x']);
        $create->assertStatus(201);
        $this->assertSame('', $create->json('data.description'));
    }

    // A present-but-null description (valid JSON, distinct from an omitted
    // key) must stay in the D10 envelope as validation_error, not 500 — the
    // same contract benchmarks/apps/rails settled on review, and what
    // benchmarks/apps/django enforces. Here the `string` rule rejects a null
    // value before it can reach the NOT NULL column.
    public function test_rejects_null_description_on_create(): void
    {
        $owner = $this->makeOwner();
        $response = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'x', 'description' => null]);
        $response->assertStatus(422);
        $this->assertSame('validation_error', $response->json('error.code'));
    }

    public function test_rejects_null_description_on_update_without_partially_applying(): void
    {
        $owner = $this->makeOwner();
        $id = $this->postJson('/api/projects', ['owner_id' => $owner->id, 'name' => 'x', 'description' => 'keep'])->json('data.id');

        $response = $this->patchJson("/api/projects/{$id}", ['description' => null]);
        $response->assertStatus(422);
        $this->assertSame('validation_error', $response->json('error.code'));

        $this->assertSame('keep', $this->getJson("/api/projects/{$id}")->json('data.description'));
    }

    public function test_get_nonexistent_id_is_not_found(): void
    {
        $response = $this->getJson('/api/projects/999999999');
        $response->assertStatus(404);
        $this->assertSame('not_found', $response->json('error.code'));
    }

    // A route parameter that isn't an id at all must still get the D10
    // not_found envelope, not a raw Postgres type-cast 500 — see
    // ProjectController::findProject.
    public function test_get_non_numeric_id_is_not_found(): void
    {
        $response = $this->getJson('/api/projects/not-a-number');
        $response->assertStatus(404);
        $this->assertSame('not_found', $response->json('error.code'));
    }

    private function seedFixture(int $userCount, int $projectCount): void
    {
        for ($i = 1; $i <= $userCount; $i++) {
            User::create(['email' => "fixture-{$i}@example.com", 'name' => "Fixture User {$i}"]);
        }
        $owners = User::orderBy('id')->pluck('id')->all();
        for ($i = 1; $i <= $projectCount; $i++) {
            Project::create([
                'owner_id' => $owners[CanonicalSeed::projectOwnerId($i, $userCount) - 1],
                'name' => "Fixture Project {$i}",
                'description' => 'fixture',
            ]);
        }
    }

    public function test_list_pagination_and_ordering(): void
    {
        $this->seedFixture(3, 25);

        $page1 = $this->getJson('/api/projects?page=1&limit=20')->json();
        $this->assertSame(['page' => 1, 'limit' => 20, 'total' => 25], $page1['meta']);
        $this->assertCount(20, $page1['data']);
        $this->assertGreaterThan($page1['data'][1]['id'], $page1['data'][0]['id']);
        foreach ($page1['data'] as $row) {
            $this->assertNotSame('', $row['owner_name']);
        }

        $page2 = $this->getJson('/api/projects?page=2&limit=20')->json();
        $this->assertCount(5, $page2['data']);
        $this->assertGreaterThan($page2['data'][0]['id'], $page1['data'][19]['id']);
    }

    // benchmarks/docs/schema.md "Canonical CRUD API": a fixed number of
    // queries independent of page size. Project::with('owner') preloads
    // owners via one batched `where id in (...)`, matching gin-gorm's pinned
    // shape exactly: 3 queries for a non-empty page (COUNT, page SELECT,
    // batched owner SELECT), 2 for an empty one (no owners to preload).
    public function test_list_does_not_n_plus_1(): void
    {
        $this->seedFixture(5, 20);
        DB::flushQueryLog();
        DB::enableQueryLog();
        $this->getJson('/api/projects?page=1&limit=20')->assertStatus(200);
        $this->assertCount(3, DB::getQueryLog());
    }

    public function test_list_does_not_n_plus_1_on_empty_page(): void
    {
        $this->seedFixture(5, 20);
        DB::flushQueryLog();
        DB::enableQueryLog();
        $this->getJson('/api/projects?page=99&limit=20')->assertStatus(200);
        $this->assertCount(2, DB::getQueryLog());
    }
}
