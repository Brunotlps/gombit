import { Column, Entity, OneToMany, PrimaryGeneratedColumn } from 'typeorm';
import { Project } from './project.entity';
import { isoTimestamp } from './iso-timestamp.transformer';

// benchmarks/docs/schema.md "Tables" — users. Columns are `text` (unbounded,
// matching the canonical schema and every sibling), not TypeORM's default
// varchar. Only created_at (no updated_at) — the canonical users table has a
// single timestamp. `id` is bigint, which TypeORM surfaces as a string in JS;
// the controller's serializer converts it to a number for the response.
@Entity({ name: 'users' })
export class User {
  @PrimaryGeneratedColumn({ type: 'bigint' })
  id!: string;

  @Column({ type: 'text', unique: true })
  email!: string;

  @Column({ type: 'text' })
  name!: string;

  // ISO-8601 string with microseconds (see isoTimestamp / data-source.ts);
  // the DB default now() sets it on insert.
  @Column({ type: 'timestamptz', precision: 6, transformer: isoTimestamp })
  created_at!: string;

  @OneToMany(() => Project, (project) => project.owner)
  projects!: Project[];
}
