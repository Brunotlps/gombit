import { UnprocessableEntityException } from '@nestjs/common';
import { ValidationError } from 'class-validator';

// Turns class-validator's errors into a D10 `fields` map and throws a 422
// (not NestJS's default 400 for a ValidationPipe failure) — D10's
// validation_error category is always 422, and issue #141 §15 wants
// equivalent invalid input rejected the same way across implementations.
export function d10ValidationException(errors: ValidationError[]): UnprocessableEntityException {
  const fields: Record<string, string[]> = {};
  for (const error of errors) {
    fields[error.property] = Object.values(error.constraints ?? {});
  }
  return new UnprocessableEntityException({ message: 'invalid request body', d10Fields: fields });
}
