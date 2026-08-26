// issue #141's recommended initial dataset size (benchmarks/docs/schema.md
// "Seed dataset"), matching benchmarks/apps/shared.SeedUserCount /
// SeedProjectCount exactly — this app can't import that Go package (or the
// Django/Rails/Laravel ports), so the numbers and formulas are duplicated by
// hand here and must be kept in sync with them manually.
export const USER_COUNT = 1000;
export const PROJECT_COUNT = 100000;

// Rows per INSERT — an implementation detail of the seeder, not part of the
// cross-implementation seed contract (mirrors gin-gorm's seedBatchSize etc.).
export const BATCH_SIZE = 1000;

function pad(n: number, width: number): string {
  return n.toString().padStart(width, '0');
}

export function userEmail(i: number): string {
  return `user-${pad(i, 4)}@example.com`;
}

export function userName(i: number): string {
  return `User ${pad(i, 4)}`;
}

// 1-based owner id for project i, round-robin over 1..userCount, so every user
// owns projectCount/userCount projects and two implementations' seeded row N
// are content-identical. Port of benchmarks/apps/shared.ProjectOwnerID.
export function projectOwnerId(i: number, userCount: number): number {
  if (i < 1 || userCount < 1) {
    throw new Error('i and userCount must be positive');
  }
  return ((i - 1) % userCount) + 1;
}

export function projectName(i: number): string {
  return `Project ${pad(i, 6)}`;
}

export function projectDescription(i: number): string {
  return `Seeded benchmark project ${pad(i, 6)}`;
}
