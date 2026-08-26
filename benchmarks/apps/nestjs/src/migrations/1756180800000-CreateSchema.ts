import { MigrationInterface, QueryRunner } from 'typeorm';

// benchmarks/docs/schema.md "Tables". Written as explicit SQL rather than
// generated from the entities so the exact DDL the canonical schema requires
// is controlled here: `text` columns (not varchar), `timestamptz(6)` with a
// DB-side `now()` default (so writes carry microseconds without a JS Date,
// which is millisecond-only), and a plain, immediately-checked, non-deferrable
// foreign key.
export class CreateSchema1756180800000 implements MigrationInterface {
  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`
      CREATE TABLE "users" (
        "id" bigserial PRIMARY KEY,
        "email" text NOT NULL,
        "name" text NOT NULL,
        "created_at" timestamptz(6) NOT NULL DEFAULT now()
      )
    `);
    await queryRunner.query(`CREATE UNIQUE INDEX "users_email_key" ON "users" ("email")`);

    await queryRunner.query(`
      CREATE TABLE "projects" (
        "id" bigserial PRIMARY KEY,
        "owner_id" bigint NOT NULL,
        "name" text NOT NULL,
        "description" text NOT NULL DEFAULT '',
        "created_at" timestamptz(6) NOT NULL DEFAULT now(),
        "updated_at" timestamptz(6) NOT NULL DEFAULT now(),
        CONSTRAINT "projects_owner_id_fkey" FOREIGN KEY ("owner_id") REFERENCES "users" ("id")
      )
    `);
    await queryRunner.query(`CREATE INDEX "projects_owner_id_idx" ON "projects" ("owner_id")`);
    await queryRunner.query(`CREATE INDEX "projects_created_at_idx" ON "projects" ("created_at")`);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`DROP TABLE "projects"`);
    await queryRunner.query(`DROP TABLE "users"`);
  }
}
