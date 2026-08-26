# Seed the deterministic BENCH-1 benchmark dataset (1,000 users, 100,000
# projects), truncating first. Run with `bin/rails db:seed`.
CanonicalSeed.seed_database
puts "rails: seed complete"
