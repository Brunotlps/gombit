<?php

namespace Tests\Feature;

use App\Models\Project;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Tests\TestCase;

// benchmarks/docs/schema.md requires TIMESTAMPTZ (microsecond) columns and an
// immediately-checked, non-deferrable foreign key. Those depend on the
// migrations using timestampTz(..., 6) and a plain foreign() with no
// deferrable option — properties none of the CRUD/list tests would notice if
// they regressed (a list/CRUD suite passes just as well on `timestamp without
// time zone` or a deferred FK). This queries information_schema /
// pg_constraint directly, the same DB-backed guard benchmarks/apps/rails
// added after review.
//
// The column DDL and the ORM write format are two independent invariants:
// the column can be timestamptz(6) while Eloquent still writes whole-second
// timestamps into it (the .000000 bug). test_...microsecond_precision below
// pins only the column; test_eloquent_writes_microsecond_timestamps pins the
// write path (the models' $dateFormat) — remove that $dateFormat and the
// former still passes while the latter fails, which is the point.
class SchemaContractTest extends TestCase
{
    use RefreshDatabase;

    public function test_timestamps_columns_are_timestamptz_with_microsecond_precision(): void
    {
        $columns = [
            ['users', 'created_at'],
            ['projects', 'created_at'],
            ['projects', 'updated_at'],
        ];

        foreach ($columns as [$table, $column]) {
            $row = DB::selectOne(
                'SELECT data_type, datetime_precision FROM information_schema.columns WHERE table_name = ? AND column_name = ?',
                [$table, $column],
            );
            $this->assertNotNull($row, "{$table}.{$column} not found");
            $this->assertSame('timestamp with time zone', $row->data_type, "{$table}.{$column} data_type");
            $this->assertSame(6, (int) $row->datetime_precision, "{$table}.{$column} precision");
        }
    }

    // Pins the ORM *write path*, not the column: Eloquent's Postgres grammar
    // formats timestamps as 'Y-m-d H:i:s' (whole seconds) unless the model
    // sets $dateFormat with microseconds, so an API-created row would store
    // .000000 even into a timestamptz(6) column — a runtime bug the DDL check
    // above cannot see. Freezes now() to a value with a known, non-zero
    // microsecond component and asserts it survives the round trip to
    // Postgres, read back as the raw stored text (independent of how Eloquent
    // parses it on read). Remove Project::$dateFormat and this fails while
    // the precision test above still passes.
    public function test_eloquent_writes_microsecond_timestamps(): void
    {
        $owner = User::create(['email' => 'o@example.com', 'name' => 'O']);

        $this->travelTo(Carbon::parse('2026-01-02 03:04:05.123456+00:00'));
        $project = Project::create(['owner_id' => $owner->id, 'name' => 'x', 'description' => '']);
        $this->travelBack();

        $raw = DB::selectOne(
            'SELECT created_at::text AS created_at, updated_at::text AS updated_at FROM projects WHERE id = ?',
            [$project->id],
        );
        $this->assertStringContainsString('.123456', $raw->created_at, 'created_at lost microseconds on write');
        $this->assertStringContainsString('.123456', $raw->updated_at, 'updated_at lost microseconds on write');
    }

    public function test_owner_foreign_key_is_not_deferrable(): void
    {
        $row = DB::selectOne(
            "SELECT condeferrable, condeferred FROM pg_constraint WHERE conrelid = 'projects'::regclass AND contype = 'f'",
        );
        $this->assertNotNull($row, 'projects.owner_id foreign key not found');
        $this->assertFalse((bool) $row->condeferrable, 'FK must not be DEFERRABLE');
        $this->assertFalse((bool) $row->condeferred, 'FK must not be INITIALLY DEFERRED');
    }
}
