require "test_helper"

# Pure, no-database checks of the seed content formulas — port of
# benchmarks/apps/shared/seed_test.go's
# TestSeedContentIsDeterministic/TestProjectOwnerIDRoundRobin (also ported
# to benchmarks/apps/django's SeedContentTests). This app can't import
# either, so the same properties are re-verified against this app's own
# from-scratch port instead of merely trusted.
class CanonicalSeedTest < ActiveSupport::TestCase
  test "seed content is deterministic" do
    assert_equal "user-0001@example.com", CanonicalSeed.user_email(1)
    assert_equal "User 0001", CanonicalSeed.user_name(1)
    assert_equal "user-1000@example.com", CanonicalSeed.user_email(1000)
    assert_equal "Project 000001", CanonicalSeed.project_name(1)
    assert_equal "Seeded benchmark project 000001", CanonicalSeed.project_description(1)
    assert_equal "Project 100000", CanonicalSeed.project_name(100_000)
  end

  test "project_owner_id round robins" do
    user_count = 7
    assert_equal 1, CanonicalSeed.project_owner_id(1, user_count)
    assert_equal 7, CanonicalSeed.project_owner_id(7, user_count)
    # The 8th project wraps back to owner 1 — the round-robin boundary an
    # off-by-one would break silently.
    assert_equal 1, CanonicalSeed.project_owner_id(8, user_count)
    assert_equal 7, CanonicalSeed.project_owner_id(14, user_count)
    assert_equal 1, CanonicalSeed.project_owner_id(15, user_count)
  end
end
