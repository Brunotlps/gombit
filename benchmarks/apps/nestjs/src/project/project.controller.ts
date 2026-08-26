import {
  Body,
  Controller,
  Delete,
  Get,
  HttpCode,
  NotFoundException,
  Param,
  Patch,
  Post,
  Query,
} from '@nestjs/common';
import { CreateProjectDto, UpdateProjectDto } from './dto';
import { ProjectService } from './project.service';

// Canonical /api/projects CRUD API (benchmarks/docs/schema.md). The `api`
// prefix is set globally in main.ts.
@Controller('projects')
export class ProjectController {
  constructor(private readonly service: ProjectService) {}

  @Get()
  list(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.service.list(
      this.clampPage(this.toInt(page, 1)),
      this.clampLimit(this.toInt(limit, 20)),
    );
  }

  @Get(':id')
  async get(@Param('id') id: string) {
    return { data: await this.service.get(this.parseId(id)) };
  }

  @Post()
  @HttpCode(201)
  async create(@Body() dto: CreateProjectDto) {
    return { data: await this.service.create(dto) };
  }

  @Patch(':id')
  async update(@Param('id') id: string, @Body() dto: UpdateProjectDto) {
    return { data: await this.service.update(this.parseId(id), dto) };
  }

  @Delete(':id')
  async remove(@Param('id') id: string) {
    await this.service.remove(this.parseId(id));
    return { data: { ok: true } };
  }

  // A route parameter that isn't a positive integer must get the D10
  // not_found envelope, not a raw Postgres type-cast error — the same case
  // django's <str:pk> / rails's find_project! / laravel's findProject handle.
  private parseId(id: string): number {
    if (!/^\d+$/.test(id)) {
      throw new NotFoundException();
    }
    return Number(id);
  }

  private toInt(value: string | undefined, fallback: number): number {
    if (value === undefined) {
      return fallback;
    }
    const parsed = Number.parseInt(value, 10);
    return Number.isNaN(parsed) ? fallback : parsed;
  }

  private clampPage(page: number): number {
    return page < 1 ? 1 : page;
  }

  private clampLimit(limit: number): number {
    if (limit < 1) {
      return 20;
    }
    return limit > 100 ? 100 : limit;
  }
}
