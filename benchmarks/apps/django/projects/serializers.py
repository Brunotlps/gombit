from rest_framework import serializers


class ProjectDataSerializer(serializers.Serializer):
    """Canonical /api/projects response shape (benchmarks/docs/schema.md
    "Canonical CRUD API"). owner_name is the preloaded relationship
    flattened into the response — its presence is what the N+1 fairness
    check verifies (views.py's select_related("owner") is what keeps
    reading it from causing a second query per row).
    """

    id = serializers.IntegerField()
    owner_id = serializers.IntegerField()
    owner_name = serializers.CharField(source="owner.name")
    name = serializers.CharField()
    description = serializers.CharField()
    created_at = serializers.DateTimeField()
    updated_at = serializers.DateTimeField()


def _reject_blank(value):
    if not value.strip():
        raise serializers.ValidationError("name must not be blank")
    return value


class CreateProjectSerializer(serializers.Serializer):
    # min_value=1 rejects owner_id:0 — a present, well-typed field, not a
    # missing one — at validation, the same input
    # benchmarks/apps/gin-gorm's binding:"required" rejects (Gin's
    # "required" treats a non-pointer uint zero value as absent) and
    # benchmarks/apps/gombit's minimum:"1" was added to close after it was
    # initially missed there. This app gets it right from the start.
    owner_id = serializers.IntegerField(min_value=1)
    # trim_whitespace=False: this app must not silently alter a
    # client-provided name (DRF's CharField trims by default), matching
    # every other implementation, none of which trims. allow_blank=True so
    # both "" and a whitespace-only string reach validate_name and get the
    # same rejection message instead of DRF's own "may not be blank"
    # short-circuiting just the empty-string case.
    name = serializers.CharField(max_length=255, allow_blank=True, trim_whitespace=False)
    # trim_whitespace=False here too: description is a canonical stored
    # field (benchmarks/docs/schema.md), not decoration — DRF's default
    # would silently store "hello" for a client-supplied "  hello  " here
    # while gin-gorm/gombit store it byte-for-byte. Found on review: the
    # comment on `name` above stated the "don't alter client-provided
    # strings" invariant but wasn't applied to the other free-text field on
    # the same serializer.
    description = serializers.CharField(
        required=False, allow_blank=True, trim_whitespace=False, default=""
    )

    def validate_name(self, value):
        return _reject_blank(value)


class UpdateProjectSerializer(serializers.Serializer):
    """PATCH body: both fields optional, only supplied ones are applied
    (views.py checks `in serializer.validated_data`) — mirrors
    benchmarks/apps/gin-gorm's *string PATCH semantics without a nullable
    sentinel value.
    """

    name = serializers.CharField(
        max_length=255, required=False, allow_blank=True, trim_whitespace=False
    )
    description = serializers.CharField(
        required=False, allow_blank=True, trim_whitespace=False
    )

    def validate_name(self, value):
        return _reject_blank(value)
