// Side-effect import: registers the pg timestamptz->string type parser
// (data-source.ts) so this standalone DataSource, which doesn't go through
// the app module, also reads timestamps as raw strings the serializer can
// format. Jest isolates module state per test file, so it must be imported
// here too.
import '../src/data-source';
import { DataSource, Logger } from 'typeorm';
import { Project } from '../src/entities/project.entity';
import { User } from '../src/entities/user.entity';
import { ProjectService } from '../src/project/project.service';
import { projectOwnerId } from '../src/seed-formulas';

// Counts the SQL statements TypeORM issues, to pin the list endpoint's query
// shape (benchmarks/docs/schema.md): a fixed number independent of page size,
// not one owner query per row.
class CountingLogger implements Logger {
  queries: string[] = [];
  reset(): void {
    this.queries = [];
  }
  logQuery(query: string): void {
    this.queries.push(query);
  }
  logQueryError(): void {}
  logQuerySlow(): void {}
  logSchemaBuild(): void {}
  logMigration(): void {}
  log(): void {}
}

describe('list query count', () => {
  let dataSource: DataSource;
  let logger: CountingLogger;
  let service: ProjectService;

  beforeAll(async () => {
    logger = new CountingLogger();
    dataSource = new DataSource({
      type: 'postgres',
      url:
        process.env.DATABASE_URL ??
        'postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs_test?sslmode=disable',
      entities: [User, Project],
      synchronize: false,
      logger,
      logging: ['query'],
    });
    await dataSource.initialize();
    service = new ProjectService(dataSource.getRepository(Project));
  });

  afterAll(async () => {
    await dataSource.destroy();
  });

  beforeEach(async () => {
    await dataSource.query('TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE');
    const userCount = 5;
    for (let i = 1; i <= userCount; i++) {
      await dataSource.query('INSERT INTO users (email, name) VALUES ($1, $2)', [
        `u${i}@example.com`,
        `User ${i}`,
      ]);
    }
    for (let i = 1; i <= 20; i++) {
      await dataSource.query(`INSERT INTO projects (owner_id, name, description) VALUES ($1, $2, '')`, [
        projectOwnerId(i, userCount),
        `Project ${i}`,
      ]);
    }
  });

  // 3 queries for a non-empty page: COUNT, the page SELECT, and one batched
  // owner SELECT (relationLoadStrategy 'query') — matching gin-gorm's pinned
  // shape exactly.
  it('issues exactly 3 queries for a non-empty page', async () => {
    logger.reset();
    await service.list(1, 20);
    expect(logger.queries).toHaveLength(3);
  });

  // 2 for an empty page: no rows, so no owners to load.
  it('issues exactly 2 queries for an empty page', async () => {
    logger.reset();
    await service.list(99, 20);
    expect(logger.queries).toHaveLength(2);
  });
});
