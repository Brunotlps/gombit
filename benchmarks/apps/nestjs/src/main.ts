import 'reflect-metadata';
import { ValidationPipe } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { D10ExceptionFilter } from './d10/d10-exception.filter';
import { d10ValidationException } from './d10/validation.factory';

async function bootstrap(): Promise<void> {
  // `/livez` (HealthController) is registered before setGlobalPrefix's
  // exclusion, so the api prefix applies to everything except it.
  const app = await NestFactory.create(AppModule, {
    // issue #141 §19: no per-request access logging. NestFactory's default
    // logger prints startup/errors but not a line per request; the router's
    // request logging is off by default. Kept at the default (no verbose
    // per-request logging enabled).
  });

  app.setGlobalPrefix('api', { exclude: ['livez'] });
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      // Turn a class-validator failure into a D10 422 validation_error with a
      // fields map, instead of NestJS's default 400 {message,error} shape.
      exceptionFactory: d10ValidationException,
    }),
  );
  app.useGlobalFilters(new D10ExceptionFilter());

  await app.listen(Number(process.env.PORT ?? 8085), '0.0.0.0');
}

void bootstrap();
