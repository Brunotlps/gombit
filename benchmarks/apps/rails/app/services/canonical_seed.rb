# issue #141's recommended initial dataset size
# (benchmarks/docs/schema.md "Seed dataset"), matching
# benchmarks/apps/shared.SeedUserCount/SeedProjectCount exactly — this app
# can't import that Go package (or benchmarks/apps/django's Python port),
# so the numbers and formulas are duplicated by hand here and must be kept
# in sync with them manually.
module CanonicalSeed
  USER_COUNT = 1000
  PROJECT_COUNT = 100_000

  # How many rows per INSERT — an implementation detail of this seeder, not
  # part of the cross-implementation seed contract (mirrors
  # benchmarks/apps/gin-gorm's seedBatchSize /
  # benchmarks/apps/django's SEED_BATCH_SIZE).
  BATCH_SIZE = 1000

  module_function

  def user_email(i) = format("user-%04d@example.com", i)
  def user_name(i) = format("User %04d", i)

  # 1-based owner id for project i, round-robin over 1..user_count, so
  # every user owns project_count/user_count projects and two
  # implementations' seeded row N are content-identical. Port of
  # benchmarks/apps/shared.ProjectOwnerID — same formula, independently
  # implemented (that package is Go-only).
  def project_owner_id(i, user_count)
    raise ArgumentError, "i and user_count must be positive" if i < 1 || user_count < 1
    (i - 1) % user_count + 1
  end

  def project_name(i) = format("Project %06d", i)
  def project_description(i) = format("Seeded benchmark project %06d", i)

  # Truncates and repopulates the canonical benchmark dataset at the given
  # scale. seed_database always calls this with the canonical
  # USER_COUNT/PROJECT_COUNT; tests call it directly with small counts so
  # they don't pay for a 100,000-row insert.
  #
  # Truncating first (RESTART IDENTITY) makes repeated invocations
  # idempotent instead of accumulating duplicate data, and resets the users
  # sequence to 1 so project_owner_id's round-robin computation stays
  # correct without reading back generated ids — mirrors
  # benchmarks/apps/gin-gorm's seedDatabaseN exactly. insert_all, not
  # create!, for the same reason gin-gorm batches with plain Create and
  # benchmarks/apps/django uses bulk_create: skips per-row
  # validation/callback overhead for a bulk load that's already known-valid
  # by construction — but unlike create!, insert_all does not set
  # timestamps automatically, so they're included in every row hash below.
  def seed_database_n(user_count, project_count)
    ActiveRecord::Base.connection.execute("TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE")

    (1..user_count).each_slice(BATCH_SIZE) do |batch|
      rows = batch.map { |i| { email: user_email(i), name: user_name(i), created_at: Time.current } }
      User.insert_all(rows)
    end

    (1..project_count).each_slice(BATCH_SIZE) do |batch|
      now = Time.current
      rows = batch.map do |i|
        {
          owner_id: project_owner_id(i, user_count),
          name: project_name(i),
          description: project_description(i),
          created_at: now,
          updated_at: now
        }
      end
      Project.insert_all(rows)
    end
  end

  # BENCH_SEED_USERS / BENCH_SEED_PROJECTS override the row counts for the CI
  # smoke's small deterministic seed (issue #141 §11) — same content formulas,
  # fewer rows. Unset/empty means the canonical default; a non-empty value that
  # is not a positive integer is a fatal error, never a silent fall back to
  # 100k. Same names and semantics as benchmarks/apps/shared.SeedCounts (Go).
  def seed_count(env, default)
    raw = ENV[env].to_s.strip
    return default if raw.empty?
    raise ArgumentError, "#{env}=#{raw.inspect}: must be a positive integer" unless raw.match?(/\A[1-9]\d*\z/)
    raw.to_i
  end

  def seed_database
    seed_database_n(
      seed_count("BENCH_SEED_USERS", USER_COUNT),
      seed_count("BENCH_SEED_PROJECTS", PROJECT_COUNT),
    )
  end
end
