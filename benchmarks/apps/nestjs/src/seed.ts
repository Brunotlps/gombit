import { AppDataSource } from './data-source';
import {
  BATCH_SIZE,
  PROJECT_COUNT,
  USER_COUNT,
  projectDescription,
  projectName,
  projectOwnerId,
  userEmail,
  userName,
} from './seed-formulas';
import { DataSource } from 'typeorm';

// Truncates and repopulates the canonical benchmark dataset at the given
// scale. seedDatabase() always uses the canonical USER_COUNT/PROJECT_COUNT;
// tests call seedDatabaseN directly with small counts. Truncating first
// (RESTART IDENTITY) makes repeated runs idempotent and resets the users
// sequence to 1 so projectOwnerId's round-robin stays correct without reading
// back generated ids — mirrors gin-gorm's seedDatabaseN. Raw INSERTs (query
// builder) skip entity overhead and let the DB default now() set the
// microsecond timestamps.
export async function seedDatabaseN(
  dataSource: DataSource,
  userCount: number,
  projectCount: number,
): Promise<void> {
  await dataSource.query('TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE');

  for (let start = 1; start <= userCount; start += BATCH_SIZE) {
    const end = Math.min(start + BATCH_SIZE - 1, userCount);
    const values: unknown[] = [];
    const rows: string[] = [];
    for (let i = start; i <= end; i++) {
      const base = values.length;
      rows.push(`($${base + 1}, $${base + 2})`);
      values.push(userEmail(i), userName(i));
    }
    await dataSource.query(`INSERT INTO users (email, name) VALUES ${rows.join(',')}`, values);
  }

  for (let start = 1; start <= projectCount; start += BATCH_SIZE) {
    const end = Math.min(start + BATCH_SIZE - 1, projectCount);
    const values: unknown[] = [];
    const rows: string[] = [];
    for (let i = start; i <= end; i++) {
      const base = values.length;
      rows.push(`($${base + 1}, $${base + 2}, $${base + 3})`);
      values.push(projectOwnerId(i, userCount), projectName(i), projectDescription(i));
    }
    await dataSource.query(
      `INSERT INTO projects (owner_id, name, description) VALUES ${rows.join(',')}`,
      values,
    );
  }
}

// BENCH_SEED_USERS / BENCH_SEED_PROJECTS override the row counts for the CI
// smoke's small deterministic seed (issue #141 §11) — same content formulas,
// fewer rows. Unset/empty means the canonical default; a non-empty value that
// is not a positive integer is a fatal error, never a silent fall back to 100k.
// Same names and semantics as benchmarks/apps/shared.SeedCounts (Go).
function seedCount(env: string, def: number): number {
  const raw = (process.env[env] ?? '').trim();
  if (raw === '') {
    return def;
  }
  if (!/^[1-9]\d*$/.test(raw)) {
    throw new Error(`${env}=${raw}: must be a positive integer`);
  }
  return Number(raw);
}

async function main(): Promise<void> {
  const userCount = seedCount('BENCH_SEED_USERS', USER_COUNT);
  const projectCount = seedCount('BENCH_SEED_PROJECTS', PROJECT_COUNT);
  await AppDataSource.initialize();
  try {
    await seedDatabaseN(AppDataSource, userCount, projectCount);
    console.log('nestjs: seed complete');
  } finally {
    await AppDataSource.destroy();
  }
}

if (require.main === module) {
  void main();
}
