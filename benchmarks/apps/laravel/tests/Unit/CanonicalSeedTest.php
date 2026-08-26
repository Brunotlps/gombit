<?php

namespace Tests\Unit;

use App\Support\CanonicalSeed;
use PHPUnit\Framework\TestCase;

// Pure, no-database checks of the seed content formulas — port of
// benchmarks/apps/shared/seed_test.go's
// TestSeedContentIsDeterministic/TestProjectOwnerIDRoundRobin (also ported to
// the Django and Rails suites). This app can't import any of them, so the
// same properties are re-verified against this app's own from-scratch port.
class CanonicalSeedTest extends TestCase
{
    public function test_seed_content_is_deterministic(): void
    {
        $this->assertSame('user-0001@example.com', CanonicalSeed::userEmail(1));
        $this->assertSame('User 0001', CanonicalSeed::userName(1));
        $this->assertSame('user-1000@example.com', CanonicalSeed::userEmail(1000));
        $this->assertSame('Project 000001', CanonicalSeed::projectName(1));
        $this->assertSame('Seeded benchmark project 000001', CanonicalSeed::projectDescription(1));
        $this->assertSame('Project 100000', CanonicalSeed::projectName(100000));
    }

    public function test_project_owner_id_round_robins(): void
    {
        $userCount = 7;
        $this->assertSame(1, CanonicalSeed::projectOwnerId(1, $userCount));
        $this->assertSame(7, CanonicalSeed::projectOwnerId(7, $userCount));
        // The 8th project wraps back to owner 1 — the round-robin boundary an
        // off-by-one would break silently.
        $this->assertSame(1, CanonicalSeed::projectOwnerId(8, $userCount));
        $this->assertSame(7, CanonicalSeed::projectOwnerId(14, $userCount));
        $this->assertSame(1, CanonicalSeed::projectOwnerId(15, $userCount));
    }
}
