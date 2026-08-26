import { IsInt, IsNotEmpty, IsString, Matches, MaxLength, ValidateIf } from 'class-validator';

// owner_id required and an integer: a missing/non-integer/null value fails
// here (422). owner_id 0 passes @IsInt and is left to the foreign-key
// constraint to reject as 422 (SQLSTATE 23503), leaning on the FK the way
// gin-gorm/django do rather than an extra existence query.
//
// name required, non-blank: @IsNotEmpty rejects "", @Matches(/\S/) rejects a
// whitespace-only string.
//
// description: @ValidateIf(present) means the @IsString rule runs only when
// the key is present — a present null fails @IsString (422), an omitted key
// is skipped (the service defaults it to ""), and a present string
// (including "") is kept verbatim. This matches the present-null contract
// benchmarks/apps/rails/django/laravel settled on. NestJS does not trim
// request strings by default, so whitespace and "" survive without any
// middleware to disable (unlike Laravel's TrimStrings).
export class CreateProjectDto {
  @IsInt()
  owner_id!: number;

  @IsString()
  @IsNotEmpty()
  @Matches(/\S/)
  @MaxLength(255)
  name!: string;

  @ValidateIf((o: CreateProjectDto) => o.description !== undefined)
  @IsString()
  description?: string;
}

export class UpdateProjectDto {
  @ValidateIf((o: UpdateProjectDto) => o.name !== undefined)
  @IsString()
  @IsNotEmpty()
  @Matches(/\S/)
  @MaxLength(255)
  name?: string;

  @ValidateIf((o: UpdateProjectDto) => o.description !== undefined)
  @IsString()
  description?: string;
}
