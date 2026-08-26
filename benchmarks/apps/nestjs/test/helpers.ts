import { INestApplication, ValidationPipe } from '@nestjs/common';
import { Test } from '@nestjs/testing';
import { DataSource } from 'typeorm';
import { AppModule } from '../src/app.module';
import { D10ExceptionFilter } from '../src/d10/d10-exception.filter';
import { d10ValidationException } from '../src/d10/validation.factory';

// Builds the app exactly as main.ts does (same prefix, ValidationPipe, and D10
// filter) so the e2e tests exercise the real request pipeline, and runs the
// migrations so the connected database (a dedicated gombit_bench_nestjs_test,
// set via DATABASE_URL) has the canonical schema.
export async function createTestApp(): Promise<{ app: INestApplication; dataSource: DataSource }> {
  const moduleRef = await Test.createTestingModule({ imports: [AppModule] }).compile();
  const app = moduleRef.createNestApplication();
  app.setGlobalPrefix('api', { exclude: ['livez'] });
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      exceptionFactory: d10ValidationException,
    }),
  );
  app.useGlobalFilters(new D10ExceptionFilter());
  await app.init();

  const dataSource = app.get(DataSource);
  await dataSource.runMigrations();
  return { app, dataSource };
}

export async function truncate(dataSource: DataSource): Promise<void> {
  await dataSource.query('TRUNCATE TABLE projects, users RESTART IDENTITY CASCADE');
}
