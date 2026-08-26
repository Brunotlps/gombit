from django.db import migrations

# benchmarks/docs/schema.md's projects.owner_id FK is a plain,
# immediately-checked foreign key (verified against benchmarks/apps/gin-gorm's
# real migration: no DEFERRABLE clause at all, which is Postgres's default
# meaning). Django's Postgres backend always emits FK constraints as
# DEFERRABLE INITIALLY DEFERRED instead (a fixed backend-level choice, not a
# per-field ForeignKey option — see projects/views.py's now-removed
# _write_project for the full story of the correctness bug that caused:
# inside any wrapping transaction, a deferred check let a foreign-key
# violation silently "succeed" past the write, only surfacing later as an
# unrelated-looking Project.DoesNotExist rather than the IntegrityError this
# app maps to a 422). Rather than compensate for that in every view (extra
# BEGIN/SET CONSTRAINTS/COMMIT on the production write path this benchmark
# measures), this migration makes the schema match the canonical contract
# directly, so the constraint is immediate everywhere — including inside a
# transaction — the same as every other implementation.
_CONSTRAINT_NAME = "projects_owner_id_fk"


def make_not_deferrable(apps, schema_editor):
    if schema_editor.connection.vendor != "postgresql":
        return
    with schema_editor.connection.cursor() as cursor:
        # The FK's auto-generated name (projects_owner_id_<hash>_fk_users_id)
        # depends on Django's own naming hash, not just the migration
        # history — looked up rather than hardcoded so this migration
        # doesn't silently no-op if that hash ever differs.
        cursor.execute(
            """
            SELECT conname FROM pg_constraint
            WHERE conrelid = 'projects'::regclass
              AND contype = 'f'
              AND conkey = (
                  SELECT array_agg(attnum) FROM pg_attribute
                  WHERE attrelid = 'projects'::regclass AND attname = 'owner_id'
              )
            """
        )
        row = cursor.fetchone()
        if row is None:
            raise RuntimeError("projects.owner_id foreign key constraint not found")
        existing_name = row[0]
        cursor.execute(f'ALTER TABLE projects DROP CONSTRAINT "{existing_name}"')
        cursor.execute(
            f'ALTER TABLE projects ADD CONSTRAINT "{_CONSTRAINT_NAME}" '
            "FOREIGN KEY (owner_id) REFERENCES users(id) NOT DEFERRABLE INITIALLY IMMEDIATE"
        )


def make_deferrable(apps, schema_editor):
    if schema_editor.connection.vendor != "postgresql":
        return
    with schema_editor.connection.cursor() as cursor:
        cursor.execute(f'ALTER TABLE projects DROP CONSTRAINT "{_CONSTRAINT_NAME}"')
        cursor.execute(
            'ALTER TABLE projects ADD CONSTRAINT "projects_owner_id_a6ce54bc_fk_users_id" '
            "FOREIGN KEY (owner_id) REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED"
        )


class Migration(migrations.Migration):
    dependencies = [("projects", "0001_initial")]

    operations = [migrations.RunPython(make_not_deferrable, make_deferrable)]
