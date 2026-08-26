require "test_helper"

# Mirrors benchmarks/apps/gin-gorm/benchmarks/apps/gombit's
# TestSeedDatabaseNIsIdempotentAndCorrect and benchmarks/apps/django's
# SeedDatabaseIsIdempotentAndCorrectTests: exercises the real
# truncate-then-seed path at a small scale, twice, to confirm it truncates
# rather than accumulates duplicate data on repeated invocations. Two
# TRUNCATEs inside one Rails-test-wrapped transaction work here without
# needing benchmarks/apps/django's TransactionTestCase workaround: this
# app's projects.owner_id foreign key is plain (not Postgres's deferred-by-
# default when Django generates it), verified immediate from the first
# migration — see ../README.md.
class SeedDatabaseTest < ActiveSupport::TestCase
  test "seed_database_n is idempotent and correct" do
    user_count, project_count = 7, 23 # not a multiple, to exercise the round-robin remainder

    [1, 2].each do |run|
      CanonicalSeed.seed_database_n(user_count, project_count)

      assert_equal user_count, User.count, "run #{run}"
      assert_equal project_count, Project.count, "run #{run}"

      first_user = User.find(1)
      assert_equal CanonicalSeed.user_email(1), first_user.email
      assert_equal CanonicalSeed.user_name(1), first_user.name

      # Project user_count+1 (8th project, user_count=7) is the first to
      # wrap back to owner 1 — the round-robin boundary a naive off-by-one
      # would break silently.
      wrapped = Project.find(user_count + 1)
      assert_equal 1, wrapped.owner_id, "run #{run}"
      assert_equal CanonicalSeed.project_name(user_count + 1), wrapped.name, "run #{run}"
    end
  end
end
