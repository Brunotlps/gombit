<?php

namespace App\Support;

use Illuminate\Support\Facades\DB;

// issue #141's recommended initial dataset size (benchmarks/docs/schema.md
// "Seed dataset"), matching benchmarks/apps/shared.SeedUserCount /
// SeedProjectCount exactly — this app can't import that Go package (or the
// Django/Rails ports), so the numbers and formulas are duplicated by hand
// here and must be kept in sync with them manually.
class CanonicalSeed
{
    public const USER_COUNT = 1000;
    public const PROJECT_COUNT = 100000;

    // Rows per INSERT — an implementation detail of this seeder, not part of
    // the cross-implementation seed contract (mirrors gin-gorm's
    // seedBatchSize / django's SEED_BATCH_SIZE / rails's BATCH_SIZE).
    public const BATCH_SIZE = 1000;

    public static function userEmail(int $i): string
    {
        return sprintf('user-%04d@example.com', $i);
    }

    public static function userName(int $i): string
    {
        return sprintf('User %04d', $i);
    }

    // 1-based owner id for project i, round-robin over 1..userCount, so every
    // user owns projectCount/userCount projects and two implementations'
    // seeded row N are content-identical. Port of
    // benchmarks/apps/shared.ProjectOwnerID.
    public static function projectOwnerId(int $i, int $userCount): int
    {
        if ($i < 1 || $userCount < 1) {
            throw new \InvalidArgumentException('i and userCount must be positive');
        }
        return ($i - 1) % $userCount + 1;
    }

    public static function projectName(int $i): string
    {
        return sprintf('Project %06d', $i);
    }

    public static function projectDescription(int $i): string
    {
        return sprintf('Seeded benchmark project %06d', $i);
    }

    // Truncates and repopulates the canonical benchmark dataset at the given
    // scale. seedDatabase() always calls this with the canonical
    // USER_COUNT/PROJECT_COUNT; tests call it directly with small counts so
    // they don't pay for a 100,000-row insert.
    //
    // Truncating first (RESTART IDENTITY) makes repeated invocations
    // idempotent instead of accumulating duplicate data, and resets the users
    // sequence to 1 so projectOwnerId's round-robin computation stays correct
    // without reading back generated ids — mirrors gin-gorm's seedDatabaseN.
    // DB::table()->insert (query builder, not Eloquent create) skips
    // per-row model events for a bulk load, so created_at/updated_at are set
    // explicitly here.
    public static function seedDatabaseN(int $userCount, int $projectCount): void
    {
        DB::statement('TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE');
        $now = now()->format('Y-m-d H:i:s.uP');

        for ($start = 1; $start <= $userCount; $start += self::BATCH_SIZE) {
            $end = min($start + self::BATCH_SIZE - 1, $userCount);
            $rows = [];
            for ($i = $start; $i <= $end; $i++) {
                $rows[] = [
                    'email' => self::userEmail($i),
                    'name' => self::userName($i),
                    'created_at' => $now,
                ];
            }
            DB::table('users')->insert($rows);
        }

        for ($start = 1; $start <= $projectCount; $start += self::BATCH_SIZE) {
            $end = min($start + self::BATCH_SIZE - 1, $projectCount);
            $rows = [];
            for ($i = $start; $i <= $end; $i++) {
                $rows[] = [
                    'owner_id' => self::projectOwnerId($i, $userCount),
                    'name' => self::projectName($i),
                    'description' => self::projectDescription($i),
                    'created_at' => $now,
                    'updated_at' => $now,
                ];
            }
            DB::table('projects')->insert($rows);
        }
    }

    // BENCH_SEED_USERS / BENCH_SEED_PROJECTS override the row counts for the CI
    // smoke's small deterministic seed (issue #141 §11) — same content
    // formulas, fewer rows. Unset/empty means the canonical default; a
    // non-empty value that is not a positive integer is a fatal error, never a
    // silent fall back to 100k. Same names and semantics as
    // benchmarks/apps/shared.SeedCounts (Go). getenv (not env()) so it reads the
    // real process environment the container is given, unaffected by config
    // caching.
    private static function seedCount(string $env, int $default): int
    {
        $raw = trim((string) getenv($env));
        if ($raw === '') {
            return $default;
        }
        if (preg_match('/^[1-9]\d*$/', $raw) !== 1) {
            throw new \InvalidArgumentException("{$env}={$raw}: must be a positive integer");
        }
        return (int) $raw;
    }

    public static function seedDatabase(): void
    {
        self::seedDatabaseN(
            self::seedCount('BENCH_SEED_USERS', self::USER_COUNT),
            self::seedCount('BENCH_SEED_PROJECTS', self::PROJECT_COUNT),
        );
    }
}
