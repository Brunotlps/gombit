import {
  ArgumentsHost,
  Catch,
  ExceptionFilter,
  HttpException,
  HttpStatus,
} from '@nestjs/common';
import { Response } from 'express';
import { QueryFailedError } from 'typeorm';

// Renders every error path in the D10 envelope (benchmarks/docs/schema.md):
// {"error": {"code", "message", "fields"?}}. Reimplemented by hand for
// NestJS — this app can't import benchmarks/apps/shared (Go-only) or reuse
// the Django/Rails/Laravel envelopes.
//
// A TypeORM QueryFailedError wraps the pg error; its SQLSTATE distinguishes a
// client-caused constraint violation from a server fault — the same policy
// gin-gorm's mapPersistError / django's _map_integrity_error use (issue #141
// §15). A malformed JSON body surfaces as a SyntaxError from Express's
// body-parser and, like every other 4xx invalid input, is mapped to a 422
// validation_error rather than the native 400.
@Catch()
export class D10ExceptionFilter implements ExceptionFilter {
  catch(exception: unknown, host: ArgumentsHost): void {
    const response = host.switchToHttp().getResponse<Response>();
    const { status, code, message, fields } = this.map(exception);
    const error: Record<string, unknown> = { code, message };
    if (fields) {
      error.fields = fields;
    }
    response.status(status).json({ error });
  }

  private map(exception: unknown): {
    status: number;
    code: string;
    message: string;
    fields?: Record<string, string[]>;
  } {
    if (exception instanceof QueryFailedError) {
      const sqlstate = (exception.driverError as { code?: string })?.code;
      switch (sqlstate) {
        case '23505':
          return { status: 409, code: 'conflict', message: 'project already exists' };
        case '23503':
          return {
            status: 422,
            code: 'validation_error',
            message: 'request references a resource that does not exist',
          };
        case '23502':
          return {
            status: 422,
            code: 'validation_error',
            message: 'a required field must not be null',
          };
        default:
          return { status: 500, code: 'internal', message: 'database error' };
      }
    }

    // Malformed JSON: body-parser throws a SyntaxError with a `status`/`type`.
    if (exception instanceof SyntaxError) {
      return { status: 422, code: 'validation_error', message: 'invalid request body' };
    }

    if (exception instanceof HttpException) {
      const status = exception.getStatus();
      const body = exception.getResponse();
      const fields =
        typeof body === 'object' && body !== null
          ? (body as { d10Fields?: Record<string, string[]> }).d10Fields
          : undefined;

      if (status === HttpStatus.NOT_FOUND) {
        return { status: 404, code: 'not_found', message: 'project not found' };
      }
      if (status === HttpStatus.CONFLICT) {
        return { status: 409, code: 'conflict', message: 'conflict' };
      }
      if (status >= 400 && status < 500) {
        // Every other 4xx (validation failure, unsupported media type, method
        // not allowed, ...) is invalid input by D10's mapping — always 422
        // validation_error, never the native 400 a ValidationPipe raises.
        return { status: 422, code: 'validation_error', message: 'invalid request body', fields };
      }
      return { status: 500, code: 'internal', message: 'internal server error' };
    }

    return { status: 500, code: 'internal', message: 'internal server error' };
  }
}
