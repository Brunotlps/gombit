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

async function main(): Promise<void> {
  await AppDataSource.initialize();
  try {
    await seedDatabaseN(AppDataSource, USER_COUNT, PROJECT_COUNT);
    console.log('nestjs: seed complete');
  } finally {
    await AppDataSource.destroy();
  }
}

if (require.main === module) {
  void main();
}
