import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { DataSource } from 'typeorm';
import { projectOwnerId } from '../src/seed-formulas';
import { createTestApp, truncate } from './helpers';

// Covers the same contract the four sibling suites pin (benchmarks/docs/
// schema.md): CRUD round trip, blank-name rejection on create and update, the
// present-null description contract, whitespace/empty-string preservation,
// zero/nonexistent owner, malformed JSON, 404 for a nonexistent and a
// non-numeric id, and list pagination/ordering. Every rejection asserts the
// D10 error.code, not just the HTTP status.
describe('projects api', () => {
  let app: INestApplication;
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

  const http = () => request(app.getHttpServer());

  async function makeOwner(): Promise<number> {
    await dataSource.query(`INSERT INTO users (email, name) VALUES ('owner@example.com', 'Owner')`);
    const [row] = await dataSource.query('SELECT id FROM users ORDER BY id DESC LIMIT 1');
    return Number(row.id);
  }

  it('does a full CRUD round trip', async () => {
    const owner = await makeOwner();

    const create = await http()
      .post('/api/projects')
      .send({ owner_id: owner, name: 'Test Project', description: 'desc' });
    expect(create.status).toBe(201);
    expect(create.body.data.name).toBe('Test Project');
    expect(create.body.data.owner_name).toBe('Owner');
    const id = create.body.data.id;

    expect((await http().get(`/api/projects/${id}`)).status).toBe(200);

    const update = await http().patch(`/api/projects/${id}`).send({ name: 'Renamed' });
    expect(update.status).toBe(200);
    expect(update.body.data.name).toBe('Renamed');
    expect(update.body.data.description).toBe('desc'); // unchanged

    expect((await http().delete(`/api/projects/${id}`)).status).toBe(200);
    expect((await http().get(`/api/projects/${id}`)).status).toBe(404);
  });

  // ProjectService.update sets updated_at to SQL now() (never a JS Date); if
  // that raw-SQL function value were ever dropped/stringified, PATCH would
  // still 200 but leave updated_at equal to created_at. gin-gorm/gombit pin
  // the equivalent (UpdatedAt not before CreatedAt after PATCH); this asserts
  // updated_at strictly advances. Timestamps are ISO strings (microsecond),
  // so lexicographic comparison is chronological.
  it('advances updated_at on update', async () => {
    const owner = await makeOwner();
    const create = await http()
      .post('/api/projects')
      .send({ owner_id: owner, name: 'x', description: 'd' });
    const created = create.body.data;
    expect(created.updated_at).toBe(created.created_at); // equal on insert

    await new Promise((resolve) => setTimeout(resolve, 10));
    const update = await http().patch(`/api/projects/${created.id}`).send({ name: 'y' });
    const updated = update.body.data;

    expect(updated.created_at).toBe(created.created_at); // created_at unchanged
    expect(updated.updated_at > updated.created_at).toBe(true);
    expect(updated.updated_at).not.toBe(created.updated_at);
  });

  it('rejects a blank name on create', async () => {
    const owner = await makeOwner();
    for (const name of ['', '   ']) {
      const res = await http().post('/api/projects').send({ owner_id: owner, name, description: 'x' });
      expect(res.status).toBe(422);
      expect(res.body.error.code).toBe('validation_error');
    }
  });

  it('rejects a blank name on update without partially applying', async () => {
    const owner = await makeOwner();
    const { body } = await http()
      .post('/api/projects')
      .send({ owner_id: owner, name: 'Original', description: 'desc' });
    const id = body.data.id;

    for (const name of ['', '   ']) {
      const res = await http().patch(`/api/projects/${id}`).send({ name });
      expect(res.status).toBe(422);
      expect(res.body.error.code).toBe('validation_error');
    }
    expect((await http().get(`/api/projects/${id}`)).body.data.name).toBe('Original');
  });

  it('rejects a zero owner_id', async () => {
    const res = await http().post('/api/projects').send({ owner_id: 0, name: 'x' });
    expect(res.status).toBe(422);
    expect(res.body.error.code).toBe('validation_error');
  });

  it('rejects a nonexistent owner_id', async () => {
    const res = await http().post('/api/projects').send({ owner_id: 999999, name: 'Orphan' });
    expect(res.status).toBe(422);
    expect(res.body.error.code).toBe('validation_error');
  });

  it('rejects malformed JSON', async () => {
    const res = await http()
      .post('/api/projects')
      .set('Content-Type', 'application/json')
      .send('{');
    expect(res.status).toBe(422);
    expect(res.body.error.code).toBe('validation_error');
  });

  it('preserves description whitespace and empty string', async () => {
    const owner = await makeOwner();
    const padded = '  padded  ';
    const create = await http().post('/api/projects').send({ owner_id: owner, name: 'x', description: padded });
    expect(create.status).toBe(201);
    expect(create.body.data.description).toBe(padded);

    const empty = await http().post('/api/projects').send({ owner_id: owner, name: 'y', description: '' });
    expect(empty.status).toBe(201);
    expect(empty.body.data.description).toBe('');
  });

  it('defaults an omitted description to empty string', async () => {
    const owner = await makeOwner();
    const create = await http().post('/api/projects').send({ owner_id: owner, name: 'x' });
    expect(create.status).toBe(201);
    expect(create.body.data.description).toBe('');
  });

  it('rejects a present-null description on create and update', async () => {
    const owner = await makeOwner();
    const create = await http().post('/api/projects').send({ owner_id: owner, name: 'x', description: null });
    expect(create.status).toBe(422);
    expect(create.body.error.code).toBe('validation_error');

    const ok = await http().post('/api/projects').send({ owner_id: owner, name: 'y', description: 'keep' });
    const id = ok.body.data.id;
    const update = await http().patch(`/api/projects/${id}`).send({ description: null });
    expect(update.status).toBe(422);
    expect(update.body.error.code).toBe('validation_error');
    expect((await http().get(`/api/projects/${id}`)).body.data.description).toBe('keep');
  });

  it('returns 404 for a nonexistent id', async () => {
    const res = await http().get('/api/projects/999999999');
    expect(res.status).toBe(404);
    expect(res.body.error.code).toBe('not_found');
  });

  it('returns 404 for a non-numeric id', async () => {
    const res = await http().get('/api/projects/not-a-number');
    expect(res.status).toBe(404);
    expect(res.body.error.code).toBe('not_found');
  });

  it('paginates and orders by id desc across a page boundary', async () => {
    const userCount = 3;
    const projectCount = 25;
    for (let i = 1; i <= userCount; i++) {
      await dataSource.query(`INSERT INTO users (email, name) VALUES ($1, $2)`, [
        `fixture-${i}@example.com`,
        `Fixture User ${i}`,
      ]);
    }
    const owners: number[] = (await dataSource.query('SELECT id FROM users ORDER BY id')).map(
      (r: { id: string }) => Number(r.id),
    );
    for (let i = 1; i <= projectCount; i++) {
      await dataSource.query(`INSERT INTO projects (owner_id, name, description) VALUES ($1, $2, 'fixture')`, [
        owners[projectOwnerId(i, userCount) - 1],
        `Fixture Project ${i}`,
      ]);
    }

    const page1 = (await http().get('/api/projects?page=1&limit=20')).body;
    expect(page1.meta).toEqual({ page: 1, limit: 20, total: 25 });
    expect(page1.data).toHaveLength(20);
    expect(page1.data[0].id).toBeGreaterThan(page1.data[1].id);
    for (const row of page1.data) {
      expect(row.owner_name).not.toBe('');
    }

    const page2 = (await http().get('/api/projects?page=2&limit=20')).body;
    expect(page2.data).toHaveLength(5);
    expect(page1.data[19].id).toBeGreaterThan(page2.data[0].id);
  });
});
