# D10 response envelope (benchmarks/docs/schema.md), reimplemented by hand
# for Rails — this app can't import benchmarks/apps/shared (Go-only) or
# reuse benchmarks/apps/django's Python envelope, so it matches the
# documented JSON shape independently, the same way
# benchmarks/apps/gin-gorm's shared.ErrorEnvelope and
# benchmarks/apps/django's projects/envelope.py both do for their own
# languages.
#
# Success: {"data": ...} for a single resource, {"data": [...], "meta": {...}}
# for the list endpoint. Error: {"error": {"code", "message", "fields"?}}.
module D10Envelope
  extend ActiveSupport::Concern

  included do
    rescue_from ActionDispatch::Http::Parameters::ParseError, with: :render_malformed_json
    rescue_from ActiveRecord::RecordNotFound, with: :render_not_found
    rescue_from ActiveRecord::RecordInvalid, with: :render_record_invalid
    rescue_from ActiveRecord::RecordNotUnique, with: :render_conflict
    rescue_from ActiveRecord::InvalidForeignKey, with: :render_invalid_foreign_key
  end

  def render_ok(data, status: :ok)
    render json: { data: data }, status: status
  end

  def render_ok_with_meta(data, meta)
    render json: { data: data, meta: meta }, status: :ok
  end

  def render_error(code, message, status, fields: nil)
    body = { code: code, message: message }
    body[:fields] = fields if fields
    render json: { error: body }, status: status
  end

  private

  def render_malformed_json(exc)
    # D10's validation_error category is always 422, not whatever native
    # 4xx Rails' own parser happened to raise for — the same fix
    # benchmarks/apps/django's envelope.exception_handler needed after
    # review (a ParseError there defaults to HTTP 400; matching it here
    # from the start rather than relearning it a third time).
    render_error("validation_error", exc.message, :unprocessable_content)
  end

  def render_not_found(_exc)
    render_error("not_found", "project not found", :not_found)
  end

  def render_record_invalid(exc)
    fields = exc.record.errors.to_hash.transform_values { |messages| messages }
    render_error("validation_error", exc.message, :unprocessable_content, fields: fields)
  end

  def render_conflict(_exc)
    render_error("conflict", "project already exists", :conflict)
  end

  def render_invalid_foreign_key(_exc)
    # A bad client-supplied reference (e.g. owner_id naming no existing
    # user) is invalid input, not a server failure — the same policy
    # benchmarks/apps/gin-gorm's mapPersistError and
    # benchmarks/apps/django's _map_integrity_error both use for the
    # equivalent foreign-key violation (issue #141 §15: equivalent invalid
    # input rejected the same way).
    render_error("validation_error", "request references a resource that does not exist", :unprocessable_content)
  end
end
