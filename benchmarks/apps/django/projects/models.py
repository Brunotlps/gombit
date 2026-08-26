from django.db import models


class User(models.Model):
    """benchmarks/docs/schema.md "Tables" — users.

    email/name are TextField, not Django's usual CharField/EmailField: the
    canonical schema is `TEXT NOT NULL` for both (verified against
    benchmarks/apps/gin-gorm and benchmarks/apps/gombit's real migrations —
    GORM's plain `string` maps to TEXT, not VARCHAR(n)), and this app never
    accepts user input for a User anyway (only seeded — see
    projects/management/commands/seed.py), so EmailField's format
    validation buys nothing a length-bounded VARCHAR wouldn't also cost in
    schema equivalence.
    """

    email = models.TextField(unique=True)
    name = models.TextField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = "users"


class Project(models.Model):
    """benchmarks/docs/schema.md "Tables" — projects.

    owner uses on_delete=DO_NOTHING rather than Django's usual CASCADE
    default: the canonical schema's FK is `ON DELETE NO ACTION` (verified
    against benchmarks/apps/gin-gorm's migration), and every implementation
    is required to express the same schema — a Django-idiomatic CASCADE
    here would silently delete a user's projects on that user's own delete,
    which no other implementation does.
    """

    # db_index=False: Meta.indexes below already adds
    # projects_owner_id_idx explicitly (named to match schema.md); without
    # this, Django would also add its own automatic same-column FK index,
    # leaving two redundant indexes on owner_id.
    owner = models.ForeignKey(
        User,
        on_delete=models.DO_NOTHING,
        db_column="owner_id",
        related_name="projects",
        db_index=False,
    )
    # TextField, not CharField(max_length=255): the canonical schema's
    # `name` column is TEXT (see User's doc comment above for the same
    # reasoning). The 255-character cap this app still enforces lives at
    # the serializer layer (serializers.py), matching every other
    # implementation's app-layer-only length validation, not a DB
    # constraint.
    name = models.TextField()
    description = models.TextField(default="", blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = "projects"
        indexes = [
            models.Index(fields=["owner"], name="projects_owner_id_idx"),
            models.Index(fields=["created_at"], name="projects_created_at_idx"),
        ]
