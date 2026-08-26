import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { Project } from '../entities/project.entity';
import { CreateProjectDto, UpdateProjectDto } from './dto';

export interface ProjectData {
  id: number;
  owner_id: number;
  owner_name: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Page {
  data: ProjectData[];
  meta: { page: number; limit: number; total: number };
}

@Injectable()
export class ProjectService {
  constructor(
    @InjectRepository(Project)
    private readonly projects: Repository<Project>,
  ) {}

  async list(page: number, limit: number): Promise<Page> {
    const total = await this.projects.count();
    const rows = await this.projects.find({
      relations: { owner: true },
      // 'query' loads owners in one batched `WHERE id IN (...)` query rather
      // than a JOIN — so the shape is COUNT + page SELECT + one owner SELECT =
      // 3 queries for a non-empty page (matching gin-gorm's pinned shape
      // exactly), and 2 for an empty page (no owners to load). A JOIN (the
      // TypeORM default) would be 2 for both — the deviation benchmarks/apps/
      // django documents; this app matches the pinned 3 instead.
      relationLoadStrategy: 'query',
      order: { id: 'DESC' },
      skip: (page - 1) * limit,
      take: limit,
    });

    return {
      data: rows.map((row) => this.serialize(row)),
      meta: { page, limit, total },
    };
  }

  async get(id: number): Promise<ProjectData> {
    return this.serialize(await this.findOrFail(id));
  }

  async create(dto: CreateProjectDto): Promise<ProjectData> {
    // created_at/updated_at are omitted so the DB defaults (now()) set them
    // with microsecond precision — no JS Date, which is millisecond-only.
    const result = await this.projects.insert({
      owner_id: String(dto.owner_id),
      name: dto.name,
      description: dto.description ?? '',
    });
    const id = Number(result.identifiers[0].id);
    return this.serialize(await this.findOrFail(id));
  }

  async update(id: number, dto: UpdateProjectDto): Promise<ProjectData> {
    await this.findOrFail(id);

    const changes: Record<string, unknown> = { updated_at: () => 'now()' };
    if (dto.name !== undefined) {
      changes.name = dto.name;
    }
    if (dto.description !== undefined) {
      changes.description = dto.description;
    }
    await this.projects.update(id, changes);
    return this.serialize(await this.findOrFail(id));
  }

  async remove(id: number): Promise<void> {
    await this.findOrFail(id);
    await this.projects.delete(id);
  }

  private async findOrFail(id: number): Promise<Project> {
    const project = await this.projects.findOne({
      where: { id: String(id) },
      relations: { owner: true },
    });
    if (!project) {
      throw new NotFoundException();
    }
    return project;
  }

  private serialize(project: Project): ProjectData {
    // created_at/updated_at are already ISO-8601 strings with microseconds —
    // the entities' isoTimestamp transformer reshapes the raw pg value at the
    // entity boundary, so the serializer does no timestamp string surgery.
    return {
      id: Number(project.id),
      owner_id: Number(project.owner_id),
      owner_name: project.owner.name,
      name: project.name,
      description: project.description,
      created_at: project.created_at,
      updated_at: project.updated_at,
    };
  }
}
