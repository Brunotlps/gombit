import {
  Column,
  Entity,
  Index,
  JoinColumn,
  ManyToOne,
  PrimaryGeneratedColumn,
} from 'typeorm';
import { User } from './user.entity';

// benchmarks/docs/schema.md "Tables" — projects. `id`/`owner_id` are bigint
// (surfaced as strings by TypeORM, converted to numbers by the serializer).
// The FK is created with no onDelete/deferrable option, so it stays
// Postgres's default NO ACTION, NOT DEFERRABLE — matching the canonical FK
// and, unlike benchmarks/apps/django's ORM, needing no follow-up migration.
// created_at/updated_at are timestamptz(6) read back as raw strings (see
// data-source.ts) and written by the DB (default now()) so both read and
// write keep microsecond precision.
@Entity({ name: 'projects' })
export class Project {
  @PrimaryGeneratedColumn({ type: 'bigint' })
  id!: string;

  @Column({ type: 'bigint' })
  @Index()
  owner_id!: string;

  @ManyToOne(() => User, (user) => user.projects)
  @JoinColumn({ name: 'owner_id' })
  owner!: User;

  @Column({ type: 'text' })
  name!: string;

  @Column({ type: 'text', default: '' })
  description!: string;

  @Column({ type: 'timestamptz', precision: 6 })
  @Index()
  created_at!: string;

  @Column({ type: 'timestamptz', precision: 6 })
  updated_at!: string;
}
