import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { DataSource } from 'typeorm';
import { createTestApp, truncate } from './helpers';

// benchmarks/docs/schema.md requires TIMESTAMPTZ(6) columns and an
// immediately-checked, non-deferrable foreign key — properties no CRUD/list
// test would notice regressing. Queried directly from information_schema /
// pg_constraint, the DB-backed guard benchmarks/apps/rails added after review.
//
// It also pins the *read path's* microsecond fidelity, which is a real
// NestJS-specific risk: the pg driver parses timestamptz into a JS Date
// (millisecond-only) by default, silently dropping microseconds against a
// timestamptz(6) column. data-source.ts overrides the pg type parser to keep
// the raw string; the last test round-trips a known .123456 through the API
// and fails if that override is removed (the column-precision assertion would
// still pass — the column stays precision 6 regardless).
describe('schema contract', () => {
  let app: INestApplication;
  let dataSource: DataSource;

  beforeAll(async () => {
    ({ app, dataSource } = await createTestApp());
  });

  afterAll(async () => {
    await app.close();
  });

  it('created_at/updated_at columns are timestamptz with microsecond precision', async () => {
    for (const [table, column] of [
      ['users', 'created_at'],
      ['projects', 'created_at'],
      ['projects', 'updated_at'],
    ]) {
      const [row] = await dataSource.query(
        'SELECT data_type, datetime_precision FROM information_schema.columns WHERE table_name = $1 AND column_name = $2',
        [table, column],
      );
      expect(row).toBeDefined();
      expect(row.data_type).toBe('timestamp with time zone');
      expect(Number(row.datetime_precision)).toBe(6);
    }
  });

  it('projects.owner_id foreign key is not deferrable', async () => {
    const [row] = await dataSource.query(
      "SELECT condeferrable, condeferred FROM pg_constraint WHERE conrelid = 'projects'::regclass AND contype = 'f'",
    );
    expect(row).toBeDefined();
    expect(row.condeferrable).toBe(false);
    expect(row.condeferred).toBe(false);
  });

  it('preserves microseconds through the read path (raw string, not a JS Date)', async () => {
    await truncate(dataSource);
    await dataSource.query(`INSERT INTO users (email, name) VALUES ('o@example.com', 'O')`);
    // Explicit microsecond timestamp; a JS Date would collapse it to .123.
    await dataSource.query(
      `INSERT INTO projects (owner_id, name, description, created_at, updated_at)
       VALUES (1, 'x', '', '2026-01-02 03:04:05.123456+00', '2026-01-02 03:04:05.123456+00')`,
    );
    const [row] = await dataSource.query('SELECT id FROM projects ORDER BY id DESC LIMIT 1');

    const res = await request(app.getHttpServer()).get(`/api/projects/${row.id}`);
    expect(res.status).toBe(200);
    expect(res.body.data.created_at).toBe('2026-01-02T03:04:05.123456Z');
    expect(res.body.data.updated_at).toBe('2026-01-02T03:04:05.123456Z');
  });
});
