-- Create "users" table
CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "email" text NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email");
-- Create "projects" table
CREATE TABLE "projects" (
  "id" bigserial NOT NULL,
  "owner_id" bigint NOT NULL,
  "name" text NOT NULL,
  "description" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_projects_owner" FOREIGN KEY ("owner_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_projects_created_at" to table: "projects"
CREATE INDEX "idx_projects_created_at" ON "projects" ("created_at");
-- Create index "idx_projects_owner_id" to table: "projects"
CREATE INDEX "idx_projects_owner_id" ON "projects" ("owner_id");
