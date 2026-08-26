require "active_support/core_ext/integer/time"

Rails.application.configure do
  # Settings specified here will take precedence over those in config/application.rb.

  # Code is not reloaded between requests.
  config.enable_reloading = false

  # Eager load code on boot for better performance and memory savings (ignored by Rake tasks).
  config.eager_load = true

  # Full error reports are disabled.
  config.consider_all_requests_local = false

  # Cache assets for far-future expiry since they are all digest stamped.
  config.public_file_server.headers = { "cache-control" => "public, max-age=#{1.year.to_i}" }

  # Enable serving of images, stylesheets, and JavaScripts from an asset server.
  # config.asset_host = "http://assets.example.com"

  # No TLS-terminating reverse proxy in front of this benchmark app — every
  # other benchmarks/apps/ implementation is also served plain HTTP
  # directly (issue #141's benchmark harness talks to each app over plain
  # HTTP; TLS termination is out of scope for what's being measured here).
  # Rails' scaffolded default (assume_ssl/force_ssl both true) would 301 or
  # reject every request from a plain-HTTP client, including the fairness
  # check and `curl`.
  config.assume_ssl = false
  config.force_ssl = false

  # Skip http-to-https redirect for the health check endpoint (inert while
  # force_ssl is false above, kept for anyone who re-enables it later).
  # config.ssl_options = { redirect: { exclude: ->(request) { request.path == "/livez" } } }

  # Log to STDOUT with the current request id as a default log tag.
  config.log_tags = [ :request_id ]
  config.logger   = ActiveSupport::TaggedLogging.logger(STDOUT)

  # issue #141 §19: per-request access logging "can massively distort
  # synthetic benchmarks" and must be disabled for every implementation
  # during a benchmark run (errors still logged). Rails' scaffolded default
  # ("info") logs a Started/Processing/Completed line for every single
  # request — verified live: every /api/projects and /livez hit produced
  # one. gin-gorm's gin.New() has no logger middleware and Django's
  # documented gunicorn command leaves --access-logfile off, so "info" here
  # would be the only implementation logging per-request by default.
  # "warn" is quiet at the request level but still surfaces real errors.
  config.log_level = ENV.fetch("RAILS_LOG_LEVEL", "warn")

  # Must name the actual health-check route (config/routes.rb defines
  # /livez, not Rails' own default /up) or this silences nothing — verified
  # live: with "/up" here, a GET /livez still produced a full
  # Started/Processing/Completed log line at "info". Kept even now that
  # log_level defaults to "warn", since RAILS_LOG_LEVEL=info is a supported
  # override for local debugging and shouldn't reintroduce a
  # health-check-per-poll log flood if used.
  config.silence_healthcheck_path = "/livez"

  # Don't log any deprecations.
  config.active_support.report_deprecations = false

  # Replace the default in-process memory cache store with a durable alternative.
  # config.cache_store = :mem_cache_store

  # Replace the default in-process and non-durable queuing backend for Active Job.
  # config.active_job.queue_adapter = :resque

  # Enable locale fallbacks for I18n (makes lookups for any locale fall back to
  # the I18n.default_locale when a translation cannot be found).
  config.i18n.fallbacks = true

  # Do not dump schema after migrations.
  config.active_record.dump_schema_after_migration = false

  # Only use :id for inspections in production.
  config.active_record.attributes_for_inspect = [ :id ]

  # RAILS_ALLOWED_HOSTS, comma-separated (default 127.0.0.1,localhost) —
  # matches benchmarks/apps/django's DJANGO_ALLOWED_HOSTS convention. Rails'
  # DNS-rebinding protection (config.hosts) rejects any request whose Host
  # header isn't in this list with a 403, before it ever reaches routing —
  # needed for local curl/fairness-check testing against 127.0.0.1.
  config.hosts = ENV.fetch("RAILS_ALLOWED_HOSTS", "127.0.0.1,localhost").split(",")
end
