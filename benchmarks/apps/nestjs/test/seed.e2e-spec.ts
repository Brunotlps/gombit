import { DataSource } from 'typeorm';
import { seedDatabaseN } from '../src/seed';
import { projectName, userEmail, userName } from '../src/seed-formulas';
import { createTestApp, truncate } from './helpers';

// Mirrors the gin-gorm/gombit/django/rails/laravel idempotency tests:
// exercises the real truncate-then-seed path at reduced scale, twice, to
// confirm it truncates rather than accumulates.
describe('seedDatabaseN', () => {
  let app: Awaited<ReturnType<typeof createTestApp>>['app'];
  let dataSource: DataSource;

  beforeAll(async () => {
    ({ app, dataSource } = await createTestApp());
  });

  afterAll(async () => {
    await app.close();
  });

  beforeEach(async () => {
    await truncate(dataSource);
  });

  it('is idempotent and correct', async () => {
    const userCount = 7;
    const projectCount = 23; // not a multiple, to exercise the round-robin remainder

    for (const run of [1, 2]) {
      await seedDatabaseN(dataSource, userCount, projectCount);

      const [{ n: users }] = await dataSource.query('SELECT count(*)::int AS n FROM users');
      const [{ n: projects }] = await dataSource.query('SELECT count(*)::int AS n FROM projects');
      expect(users).toBe(userCount);
      expect(projects).toBe(projectCount);

      const [firstUser] = await dataSource.query('SELECT email, name FROM users WHERE id = 1');
      expect(firstUser.email).toBe(userEmail(1));
      expect(firstUser.name).toBe(userName(1));

      // Project userCount+1 (8th, userCount=7) is the first to wrap back to
      // owner 1 — the round-robin boundary an off-by-one would break.
      const [wrapped] = await dataSource.query('SELECT owner_id, name FROM projects WHERE id = $1', [
        userCount + 1,
      ]);
      expect(Number(wrapped.owner_id)).toBe(1);
      expect(wrapped.name).toBe(projectName(userCount + 1));
      expect(run).toBeGreaterThan(0);
    }
  });
});
