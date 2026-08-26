import { types } from 'pg';
import { DataSource, DataSourceOptions } from 'typeorm';
import { Project } from './entities/project.entity';
import { User } from './entities/user.entity';

// The pg driver parses timestamptz/timestamp into a JS Date by default, and a
// JS Date holds only milliseconds — so a timestamptz(6) column would lose the
// microseconds the canonical schema and every sibling carry. Overriding the
// type parsers for OID 1114 (timestamp) and 1184 (timestamptz) to return the
// raw string keeps full microsecond precision on read; the entities type
// these columns as string and the serializer formats them. Set here (imported
// by both the app and the migration CLI) so it always applies.
types.setTypeParser(1114, (value) => value);
types.setTypeParser(1184, (value) => value);

export const dataSourceOptions: DataSourceOptions = {
  type: 'postgres',
  url:
    process.env.DATABASE_URL ??
    'postgres://gombit:gombit@127.0.0.1:55432/gombit_bench_nestjs?sslmode=disable',
  entities: [User, Project],
  migrations: [__dirname + '/migrations/*.{ts,js}'],
  synchronize: false,
  // No query logging (issue #141 §19) — errors still surface as thrown
  // exceptions.
  logging: false,
  // issue #141 §18 "Connection pooling": pin max open connections to 20, the
  // same ceiling every implementation uses. NestJS/Node is single-process
  // (one event loop), so this pool is the one global pool for the whole
  // server — the same single-pool topology gin-gorm/gombit's Go binary has.
  //
  // options: '-c timezone=UTC' forces every connection's session time zone to
  // UTC, so timestamptz always renders with a +00 offset (the serializer's
  // isoTimestamp transformer normalizes +00 -> Z). Without this the app would
  // inherit the server's TimeZone, and the "+00 -> Z" conversion would be a
  // coincidence rather than a guarantee.
  extra: {
    max: Number(process.env.POOL_MAX_OPEN ?? 20),
    options: '-c timezone=UTC',
  },
};

export const AppDataSource = new DataSource(dataSourceOptions);
