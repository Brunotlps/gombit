# Canonical /api/projects CRUD API (benchmarks/docs/schema.md), implemented
# idiomatically in Rails: ActiveRecord validations (see Project) do most of
# the input-rejection work a hand-rolled check does in
# benchmarks/apps/gin-gorm/gombit/django; this controller mostly wires
# params to the model and the model's own errors to the D10 envelope.
module Api
  class ProjectsController < ApplicationController
    def index
      page = clamp_page(query_int(:page, 1))
      limit = clamp_limit(query_int(:limit, 20))

      total = Project.count
      offset = (page - 1) * limit
      # includes(:owner), not references(:owner)/joins(:owner): without a
      # where/order touching the association, Rails preloads owners via one
      # batched `SELECT ... WHERE id IN (...)` query, not a JOIN — the same
      # 2-separate-queries-for-the-page shape benchmarks/apps/gin-gorm's
      # GORM .Preload("Owner") uses, so together with the COUNT above this
      # is 3 queries for a non-empty page, matching gin-gorm's pinned shape
      # exactly (verified: ProjectsControllerTest#test_list_does_not_n_plus_1).
      # An empty page needs no owners, so Rails' preloader skips that query
      # entirely — 2 total, also matching gin-gorm's empty-page shortcut.
      projects = Project.includes(:owner).order(id: :desc).offset(offset).limit(limit)

      render_ok_with_meta(projects.map { |project| serialize(project) }, { page: page, limit: limit, total: total })
    end

    def show
      render_ok(serialize(find_project!))
    end

    def create
      project = Project.new(
        owner_id: params[:owner_id],
        name: params[:name],
        description: params[:description] || ""
      )
      project.save!
      render_ok(serialize(project), status: :created)
    end

    def update
      project = find_project!
      project.name = params[:name] if params.key?(:name)
      project.description = params[:description] if params.key?(:description)
      project.save!
      render_ok(serialize(project))
    end

    def destroy
      find_project!.destroy!
      render_ok({ ok: true })
    end

    private

    # A route parameter that isn't a positive integer at all must still get
    # the D10 not_found envelope rather than a raw type-cast error from
    # Postgres (an unfiltered `Project.find("not-a-number")` raises
    # PG::InvalidTextRepresentation, not RecordNotFound) — the same case
    # benchmarks/apps/django's urls.py <str:pk> comment addresses.
    def find_project!
      raise ActiveRecord::RecordNotFound unless params[:id].match?(/\A\d+\z/)
      Project.includes(:owner).find(params[:id])
    end

    def serialize(project)
      {
        id: project.id,
        owner_id: project.owner_id,
        owner_name: project.owner.name,
        name: project.name,
        description: project.description,
        created_at: project.created_at.utc.iso8601(6),
        updated_at: project.updated_at.utc.iso8601(6)
      }
    end

    def query_int(key, default)
      raw = params[key]
      return default if raw.blank?
      Integer(raw, exception: false) || default
    end

    def clamp_page(page)
      page < 1 ? 1 : page
    end

    def clamp_limit(limit)
      return 20 if limit < 1
      return 100 if limit > 100
      limit
    end
  end
end
