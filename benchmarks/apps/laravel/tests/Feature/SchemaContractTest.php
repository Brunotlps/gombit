<?php

namespace Tests\Feature;

use Illuminate\Foundation\Testing\RefreshDatabase;
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
class SchemaContractTest extends TestCase
{
    use RefreshDatabase;

    public function test_timestamps_are_timestamptz_with_microsecond_precision(): void
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
