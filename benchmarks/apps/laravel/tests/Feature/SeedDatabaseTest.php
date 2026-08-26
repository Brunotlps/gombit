<?php

namespace Tests\Feature;

use App\Models\Project;
use App\Models\User;
use App\Support\CanonicalSeed;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

// Mirrors the gin-gorm/gombit/django/rails idempotency tests: exercises the
// real truncate-then-seed path at reduced scale, twice, to confirm it
// truncates rather than accumulates. The projects.owner_id FK is immediate
// (not deferred, unlike Django's default), so the two TRUNCATEs inside one
// RefreshDatabase-wrapped transaction don't hit the "pending trigger events"
// restriction Django's own idempotency test had to work around.
class SeedDatabaseTest extends TestCase
{
    use RefreshDatabase;

    public function test_seed_database_n_is_idempotent_and_correct(): void
    {
        $userCount = 7;
        $projectCount = 23; // not a multiple, to exercise the round-robin remainder

        foreach ([1, 2] as $run) {
            CanonicalSeed::seedDatabaseN($userCount, $projectCount);

            $this->assertSame($userCount, User::count(), "run {$run}");
            $this->assertSame($projectCount, Project::count(), "run {$run}");

            $firstUser = User::find(1);
            $this->assertSame(CanonicalSeed::userEmail(1), $firstUser->email);
            $this->assertSame(CanonicalSeed::userName(1), $firstUser->name);

            // Project userCount+1 (8th, userCount=7) is the first to wrap back
            // to owner 1 — the round-robin boundary an off-by-one would break.
            $wrapped = Project::find($userCount + 1);
            $this->assertSame(1, $wrapped->owner_id, "run {$run}");
            $this->assertSame(CanonicalSeed::projectName($userCount + 1), $wrapped->name, "run {$run}");
        }
    }
}
