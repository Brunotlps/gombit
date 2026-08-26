class Project < ApplicationRecord
  # belongs_to defaults to required (Rails 5+): validates the owner
  # association is present, which also rejects owner_id:0 or a nonexistent
  # owner_id for free (no user has id 0 or a made-up id, so the presence
  # check's SELECT finds nothing either way) — the same case
  # benchmarks/apps/gin-gorm's binding:"required" and
  # benchmarks/apps/django's serializer min_value=1 needed dedicated code
  # for. See app/controllers/concerns/d10_envelope.rb's
  # render_invalid_foreign_key for the (currently unreachable through this
  # validated path, kept for any write that bypasses validations) FK-level
  # backstop.
  belongs_to :owner, class_name: "User"

  # presence: true alone is enough here — ActiveSupport's String#blank?
  # (which the presence validator uses) already treats a whitespace-only
  # string as blank, unlike benchmarks/apps/gin-gorm's Gin binding:"required"
  # and benchmarks/apps/django's DRF CharField, both of which only reject
  # the empty string and needed a dedicated whitespace check added
  # separately.
  validates :name, presence: true, length: { maximum: 255 }
end
